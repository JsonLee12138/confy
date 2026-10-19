package confy

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/JsonLee12138/confy/provider"
	"github.com/JsonLee12138/confy/provider/local"
	"github.com/spf13/viper"
)

// Config is the main configuration instance.
type Config struct {
	v         *viper.Viper
	opts      *configOptions
	mu        *sync.RWMutex
	provider  provider.Provider
	ctx       context.Context
	cancel    context.CancelFunc
	closed    bool
	closeOnce sync.Once
	closeErr  error
	watcher   provider.Watcher
	bound     any
}

// New loads one configuration source, defaulting to local files, and enables ENV lookup.
// Explicit Provider selection conflicts with root Local options. If loading
// fails, New cancels its context and closes owned Provider resources.
func New(opts ...Option) (*Config, error) {
	o := defaultOptions()
	for _, opt := range opts {
		opt(o)
	}

	source := o.provider
	if o.providerCount > 1 || (o.providerCount > 0 && (provider.IsNilProvider(source) || o.localExplicit)) {
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

	if source == nil {
		source = local.New(o.LocalOption())
	}
	ctx, cancel := context.WithCancel(context.Background())
	v, err := source.Load(ctx)
	if err != nil || v == nil {
		cancel()
		if closer, ok := source.(io.Closer); ok {
			_ = closer.Close()
		}
		if err != nil {
			return nil, err
		}
		return nil, ErrInvalidProvider
	}
	mu := &sync.RWMutex{}
	if source, ok := source.(provider.SynchronizedProvider); ok {
		mu = source.ConfigMutex()
	}
	// Source values remain in Viper's config layer; explicit Set retains priority.
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	if o.envPrefix != "" {
		v.SetEnvPrefix(o.envPrefix)
	}
	v.AutomaticEnv()

	return &Config{
		v:        v,
		opts:     o,
		provider: source, mu: mu, ctx: ctx, cancel: cancel,
	}, nil
}

// Get returns the effective value, including ENV and explicit Set overrides.
// It remains available after Close and does not decrypt source ciphertext.
func (c *Config) Get(key string) any {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.v.Get(key)
}

// Set writes a Viper override that takes precedence over the source and ENV.
// It does not rebind targets and is ignored after Close.
func (c *Config) Set(key string, value any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return
	}
	c.v.Set(key, value)
}

// Close stops delivery and releases subscriptions and owned Provider resources.
// It does not wait for callbacks, so it may be called from a callback.
// Local native Viper watching remains process-scoped; Get remains usable.
func (c *Config) Close() error {
	if c == nil || c.mu == nil {
		return nil
	}
	c.closeOnce.Do(func() {
		c.mu.Lock()
		c.closed = true
		c.cancel()
		w := c.watcher
		c.mu.Unlock()
		if w != nil {
			c.closeErr = w.Close()
		}
		if closer, ok := c.provider.(io.Closer); ok {
			c.closeErr = errors.Join(c.closeErr, closer.Close())
		}
	})
	return c.closeErr
}
