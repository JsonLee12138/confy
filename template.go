package confy

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// resolveBaseInheritance resolves the "base" field inheritance chain.
// It loads parent configs, merges them, and returns the final flat map.
func resolveBaseInheritance(path string, fileType FileType, visited map[string]bool) (map[string]any, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	if visited[absPath] {
		return nil, ErrCycleDetected
	}
	visited[absPath] = true

	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType(string(fileType))
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("confy: failed to read %s: %w", path, err)
	}

	result := flattenViper(v)

	baseFile, ok := result["base"]
	if !ok {
		return result, nil
	}

	baseFileName, ok := baseFile.(string)
	if !ok {
		return result, nil
	}

	basePath := filepath.Join(filepath.Dir(path), baseFileName)
	if _, err := os.Stat(basePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("confy: base file not found: %s", basePath)
	}

	parent, err := resolveBaseInheritance(basePath, fileType, visited)
	if err != nil {
		return nil, err
	}

	merged := mergeConfig(parent, result)
	delete(merged, "base")

	return merged, nil
}

// mergeConfig merges base (parent) config with override (child) config.
// Child values take priority.
func mergeConfig(base, override map[string]any) map[string]any {
	merged := make(map[string]any, len(base))
	for k, v := range base {
		merged[k] = v
	}
	for k, v := range override {
		merged[k] = v
	}
	return merged
}

// flattenViper flattens a viper instance into a dot-separated key map.
func flattenViper(v *viper.Viper) map[string]any {
	result := make(map[string]any)
	for _, key := range v.AllKeys() {
		result[key] = v.Get(key)
	}
	return result
}
