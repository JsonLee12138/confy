package confy

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/spf13/viper"
)

// Config is the main configuration instance.
type Config struct {
	v         *viper.Viper
	opts      *configOptions
	mu        sync.RWMutex
	provider  Provider
	watch     watchState
	lifecycle lifecycleState
	overrides map[string]any
	bound     any
}

// New creates a new Config instance with the given options.
func New(opts ...Option) (*Config, error) {
	o := defaultOptions()
	for _, opt := range opts {
		opt(o)
	}

	provider := o.provider
	if o.providerCount > 1 || (o.providerCount > 0 && (isNilProvider(provider) || o.localExplicit)) {
		return nil, ErrInvalidOptions
	}

	// Load .env files first (lower priority than system env)
	// Auto-discovered .env files (from WithDotEnvAuto) are loaded before manually specified ones,
	// so manually specified files take higher priority.
	if o.dotEnvDir != "" {
		for _, envFile := range getDotEnvFilePaths(o.dotEnvDir) {
			if err := loadDotEnv(envFile); err != nil {
				return nil, fmt.Errorf("confy: failed to load dotenv %s: %w", envFile, err)
			}
		}
	}
	for _, envFile := range o.dotEnvFiles {
		if err := loadDotEnv(envFile); err != nil {
			return nil, fmt.Errorf("confy: failed to load dotenv %s: %w", envFile, err)
		}
	}

	if provider == nil {
		provider = &LocalProvider{opts: o}
	}
	ctx, cancel := context.WithCancel(context.Background())
	v, err := provider.Load(ctx)
	if err != nil || v == nil {
		cancel()
		if closer, ok := provider.(io.Closer); ok {
			_ = closer.Close()
		}
		if err != nil {
			return nil, err
		}
		return nil, ErrInvalidProvider
	}
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	if o.envPrefix != "" {
		v.SetEnvPrefix(o.envPrefix)
	}
	v.AutomaticEnv()

	applyEnvOverrides(v, o.envPrefix)

	return &Config{
		v:        v,
		opts:     o,
		provider: provider, lifecycle: lifecycleState{ctx: ctx, cancel: cancel}, overrides: make(map[string]any),
	}, nil
}

// Get returns the value for the given key.
func (c *Config) Get(key string) any {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.v.Get(key)
}

// Set sets a config value.
func (c *Config) Set(key string, value any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.lifecycle.closed {
		return
	}
	c.overrides[strings.ToLower(key)] = value
	c.v.Set(key, value)
}
