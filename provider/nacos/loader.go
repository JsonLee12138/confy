package nacos

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/JsonLee12138/confy/core"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/common/logger"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// Load initializes the client once, then fetches and writes the source config.
func (p *Provider) Load(ctx context.Context) (*viper.Viper, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	p.mu.Lock()
	err := p.initializeLocked()
	p.mu.Unlock()
	if err != nil {
		return nil, err
	}
	if err := p.refresh(ctx); err != nil {
		return nil, err
	}
	return p.v, nil
}

// refresh is shared by initial loading and watch notifications. Serial reads
// ensure an older SDK callback cannot overwrite a newer remote document.
func (p *Provider) refresh(ctx context.Context) error {
	p.loadMu.Lock()
	defer p.loadMu.Unlock()
	p.mu.Lock()
	content, err := p.readLocked(ctx)
	p.mu.Unlock()
	if err != nil {
		return err
	}
	return p.writeConfig(ctx, content)
}

// writeConfig replaces only Viper's source layer. ENV and explicit Set overrides
// keep their precedence, and fields removed from the source disappear.
func (p *Provider) writeConfig(ctx context.Context, content string) error {
	p.configMu.Lock()
	defer p.configMu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := p.v.ReadConfig(strings.NewReader(content)); err != nil {
		return p.safe(ErrDocument)
	}
	return nil
}

// Validate empty documents in Viper's decoder, before ReadConfig replaces the
// source map. This preserves the existing rejection rule without parsing twice
// or creating a second Viper for each update.
type documentDecoders struct{ viper.DecoderRegistry }

// Decoder wraps the registered format decoder with nonempty-document validation.
func (r documentDecoders) Decoder(format string) (viper.Decoder, error) {
	decoder, err := r.DecoderRegistry.Decoder(format)
	if err != nil {
		return nil, err
	}
	return documentDecoder{decoder}, nil
}

type documentDecoder struct{ viper.Decoder }

// Decode parses once and rejects an empty document before Viper replaces its source map.
func (d documentDecoder) Decode(content []byte, config map[string]any) error {
	if err := d.Decoder.Decode(content, config); err != nil {
		return err
	}
	if len(config) == 0 {
		return ErrDocument
	}
	return nil
}

// initializeLocked validates options and creates one owned SDK client and private cache.
// The caller holds p.mu. Initialization failures remove owned files; the SDK logger
// is silenced process-wide to prevent logging configuration content.
func (p *Provider) initializeLocked() error {
	if p.closed {
		return core.ErrClosed
	}
	if p.client != nil {
		return nil
	}
	o := p.opts
	if strings.TrimSpace(o.dataID) == "" || (len(o.servers) == 0 && o.client.Endpoint == "") || (o.format != core.YAML && o.format != core.JSON && o.format != core.TOML) || o.client.OpenKMS || o.client.TLSCfg.TrustAll {
		return ErrOptions
	}
	for _, part := range []string{o.dataID, o.group, o.client.NamespaceId} {
		if strings.ContainsAny(part, "/\\") || strings.IndexFunc(part, unicode.IsControl) >= 0 || part == "." || part == ".." {
			return ErrOptions
		}
	}
	// Never reuse a caller's snapshots/failover. Each instance gets a fresh 0700
	// child; the SDK can persist only raw source content inside this directory.
	parent := o.client.CacheDir
	if parent != "" {
		absolute, err := filepath.Abs(parent)
		if err != nil {
			return p.safe(ErrRead)
		}
		parent = absolute
		if err := os.MkdirAll(parent, 0700); err != nil {
			return p.safe(ErrRead)
		}
	}
	dir, err := os.MkdirTemp(parent, "confy-nacos-")
	if err != nil {
		return p.safe(ErrRead)
	}
	p.directory = dir
	config := o.client
	config.CacheDir = dir
	config.LogDir = filepath.Join(dir, "logs")
	if err = os.MkdirAll(config.LogDir, 0700); err != nil {
		_ = os.RemoveAll(dir)
		p.directory = ""
		return p.safe(ErrRead)
	}
	config.DisableUseSnapShot = true
	config.NotLoadCacheAtStart = true
	config.AppendToStdout = false
	config.LogLevel = "error"
	if config.TimeoutMs == 0 {
		config.TimeoutMs = 5000
	}
	// SDK logging is process-global and can print raw content even on write
	// errors. Install a non-recording SDK logger before constructing any client.
	// This also silences other SDK clients; see the package README contract.
	quietSDK.Do(func() { logger.SetLogger(&logger.NacosLogger{SugaredLogger: zap.NewNop().Sugar()}) })
	client, err := p.factory(vo.NacosClientParam{ClientConfig: &config, ServerConfigs: append([]constant.ServerConfig(nil), o.servers...)})
	if err != nil || client == nil {
		if client != nil {
			client.CloseClient()
		}
		_ = os.RemoveAll(dir)
		p.directory = ""
		return p.safe(ErrRead)
	}
	p.client = client
	return nil
}

// readLocked fetches the current document while the caller holds p.mu.
// It checks cancellation before and after the SDK request, whose timeout bounds
// in-flight work, and converts SDK failures into errors without source content.
func (p *Provider) readLocked(ctx context.Context) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if p.closed {
		return "", core.ErrClosed
	}
	content, err := p.client.GetConfig(vo.ConfigParam{DataId: p.opts.dataID, Group: p.opts.group})
	if ctx.Err() != nil {
		return "", ctx.Err()
	}
	if err != nil {
		return "", p.safe(ErrRead)
	}
	return content, nil
}
