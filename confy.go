package confy

import (
	"fmt"
	"strings"
	"sync"

	"github.com/spf13/viper"
)

// Config is the main configuration instance.
type Config struct {
	v         *viper.Viper
	opts      *configOptions
	mu        sync.RWMutex
	watchOnce sync.Once
	snapshot  map[string]any
	bound     any
}

// New creates a new Config instance with the given options.
func New(opts ...Option) (*Config, error) {
	o := defaultOptions()
	for _, opt := range opts {
		opt(o)
	}

	// Load .env files first (lower priority than system env)
	for _, envFile := range o.dotEnvFiles {
		if err := loadDotEnv(envFile); err != nil {
			return nil, fmt.Errorf("confy: failed to load dotenv %s: %w", envFile, err)
		}
	}

	configPaths := getConfigFilePaths(o)
	if o.loadAll {
		configPaths = getAllConfigFilePaths(o)
	}
	if len(configPaths) == 0 {
		return nil, fmt.Errorf("%w: in path %s", ErrNoConfigFiles, o.basePath)
	}

	v := viper.New()
	v.SetConfigType(string(o.fileType))

	for _, configPath := range configPaths {
		tempV := viper.New()
		tempV.SetConfigFile(configPath)
		if err := tempV.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("confy: error reading %s: %w", configPath, err)
		}
		for _, key := range tempV.AllKeys() {
			v.Set(key, tempV.Get(key))
		}
	}

	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	if o.envPrefix != "" {
		v.SetEnvPrefix(o.envPrefix)
	}
	v.AutomaticEnv()

	applyEnvOverrides(v, o.envPrefix)

	return &Config{
		v:    v,
		opts: o,
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
	c.v.Set(key, value)
}
