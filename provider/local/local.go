package local

import (
	"bytes"
	"context"
	"fmt"
	"sync"

	"github.com/JsonLee12138/confy/core"
	"github.com/JsonLee12138/confy/provider"
	"github.com/spf13/viper"
	"go.yaml.in/yaml/v3"
)

type Provider struct {
	loadMu   sync.Mutex
	configMu sync.RWMutex
	opts     *Options
	v        *viper.Viper
	watcher  *Watcher
}

func New(opts ...Option) *Provider {
	o := DefaultOptions()
	for _, opt := range opts {
		opt(o)
	}

	return &Provider{
		opts:    o,
		watcher: &Watcher{},
	}
}

func (*Provider) Type() provider.ProviderType {
	return provider.ProviderLocal
}

// ConfigMutex synchronizes source updates with Confy reads and binding.
func (p *Provider) ConfigMutex() *sync.RWMutex { return &p.configMu }

// Load refreshes the same Viper, preserving explicit Set overrides.
func (p *Provider) Load(ctx context.Context) (*viper.Viper, error) {
	p.loadMu.Lock()
	defer p.loadMu.Unlock()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	o := p.opts

	configPaths := getConfigFilePaths(o)
	if o.LoadAll {
		configPaths = getAllConfigFilePaths(o)
	}
	if len(configPaths) == 0 {
		return nil, fmt.Errorf("%w: in path %s", core.ErrNoConfigFiles, o.BasePath)
	}

	v := viper.New()

	for _, configPath := range configPaths {
		resolved, err := core.ResolveBaseInheritance(configPath, o.FileType, make(map[string]bool))
		if err != nil {
			return nil, fmt.Errorf("confy: failed to resolve inheritance for %s: %w", configPath, err)
		}
		for key, val := range resolved {
			v.Set(key, val)
		}
	}

	data, err := yaml.Marshal(v.AllSettings())
	if err != nil {
		return nil, err
	}
	p.configMu.Lock()
	defer p.configMu.Unlock()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if p.v == nil {
		p.v = viper.New()
	}
	// Replace the source layer, rather than accumulating Set overrides. The
	// merged document is YAML regardless of the individual source file formats.
	p.v.SetConfigType("yaml")
	err = p.v.ReadConfig(bytes.NewReader(data))
	p.v.SetConfigType(string(o.FileType))
	if err != nil {
		return nil, err
	}
	p.v.SetConfigFile(configPaths[0])
	return p.v, nil
}

// Watch starts one subscription. Later calls return the same watcher and keep
// the original context and callbacks. Viper reports native watch errors itself.
func (p *Provider) Watch(
	ctx context.Context,
	onChange func(provider.Change),
	onError func(error),
) (provider.Watcher, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if p.v == nil {
		return nil, core.ErrNotLoaded
	}

	p.watcher.start(ctx, p, onChange, onError)
	return p.watcher, nil
}
