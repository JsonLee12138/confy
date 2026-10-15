package nacos

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"unicode"

	"github.com/JsonLee12138/confy/core"
	"github.com/nacos-group/nacos-sdk-go/v2/clients"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/common/logger"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

var (
	ErrOptions   = errors.New("confy/nacos: invalid options")
	ErrRead      = errors.New("confy/nacos: remote read failed")
	ErrDocument  = errors.New("confy/nacos: expected nonempty configuration object")
	ErrSubscribe = errors.New("confy/nacos: subscription failed")
)

// sdkClient deliberately excludes publish/delete and service discovery.
type sdkClient interface {
	GetConfig(vo.ConfigParam) (string, error)
	ListenConfig(vo.ConfigParam) error
	CancelListenConfig(vo.ConfigParam) error
	CloseClient()
}

// Provider owns its SDK client and private, ephemeral SDK cache directory.
// New only collects options. Network access begins in Load.
type Provider struct {
	mu        sync.Mutex
	opts      options
	client    sdkClient
	directory string
	watcher   *watcher
	closed    bool
	factory   func(vo.NacosClientParam) (sdkClient, error)
}

var quietSDK sync.Once

// New creates a single-Data-ID provider. A provider belongs to one core.Config.
func New(opts ...Option) *Provider {
	o := options{format: core.YAML, group: "DEFAULT_GROUP"}
	for _, opt := range opts {
		opt(&o)
	}
	if o.group == "" {
		o.group = "DEFAULT_GROUP"
	}
	return &Provider{opts: o, factory: func(param vo.NacosClientParam) (sdkClient, error) { return clients.NewConfigClient(param) }}
}
func (p *Provider) resource() string {
	return "nacos://" + url.PathEscape(p.opts.client.NamespaceId) + "/" + url.PathEscape(p.opts.group) + "/" + url.PathEscape(p.opts.dataID)
}
func (p *Provider) safe(err error) error { return fmt.Errorf("%w: %s", err, p.resource()) }
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

// Load reads from the remote server. SDK timeouts bound requests that cannot
// accept Context; cancellation is checked before and after each SDK operation.
func (p *Provider) Load(ctx context.Context) (*viper.Viper, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := p.initializeLocked(); err != nil {
		return nil, err
	}
	content, err := p.readLocked(ctx)
	if err != nil {
		return nil, err
	}
	v, err := parse(content, p.opts.format)
	if err != nil {
		return nil, p.safe(ErrDocument)
	}
	return v, nil
}
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
func parse(content string, format core.FileType) (*viper.Viper, error) {
	if strings.TrimSpace(content) == "" {
		return nil, ErrDocument
	}
	v := viper.New()
	v.SetConfigType(string(format))
	if err := v.ReadConfig(bytes.NewBufferString(content)); err != nil {
		return nil, ErrDocument
	}
	if len(v.AllKeys()) == 0 {
		return nil, ErrDocument
	}
	return v, nil
}

// Close releases the owned subscription, client and ephemeral files.
func (p *Provider) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return nil
	}
	p.closed = true
	if p.watcher != nil {
		p.watcher.stopLocked()
	}
	if p.client != nil {
		p.client.CloseClient()
	}
	if p.directory != "" {
		if os.RemoveAll(p.directory) != nil {
			return p.safe(ErrRead)
		}
	}
	return nil
}

// Type identifies this adapter without making the consumer import the SDK.
func (*Provider) Type() core.ProviderType { return core.ProviderNacos }

var _ core.Provider = (*Provider)(nil)
var _ core.WatchProvider = (*Provider)(nil)
