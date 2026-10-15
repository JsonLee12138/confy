package confy

import (
	"context"
	"fmt"
	"github.com/spf13/viper"
)

// LocalProvider preserves the existing local discovery and inheritance rules.
// New selects it automatically when no provider is specified.
type LocalProvider struct {
	opts      *configOptions
	watchPath string
}

func (p *LocalProvider) Load(ctx context.Context) (*viper.Viper, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if p.opts == nil {
		p.opts = defaultOptions()
	}
	o := p.opts
	configPaths := getConfigFilePaths(o)
	if o.loadAll {
		configPaths = getAllConfigFilePaths(o)
	}
	if len(configPaths) == 0 {
		return nil, fmt.Errorf("%w: in path %s", ErrNoConfigFiles, o.basePath)
	}

	v := viper.New()
	v.SetConfigType(string(o.fileType))

	// Retain the first selected file as configuration path metadata.
	if len(configPaths) > 0 {
		v.SetConfigFile(configPaths[0])
	}

	for _, configPath := range configPaths {
		resolved, err := resolveBaseInheritance(configPath, o.fileType, make(map[string]bool))
		if err != nil {
			return nil, fmt.Errorf("confy: failed to resolve inheritance for %s: %w", configPath, err)
		}
		for key, val := range resolved {
			v.Set(key, val)
		}
	}

	if p.watchPath == "" {
		p.watchPath = configPaths[0]
	}
	return v, nil
}
