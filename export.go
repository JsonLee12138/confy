package confy

import (
	"fmt"
	"os"
	"path/filepath"
)

// Export writes the current config state to a file.
func (c *Config) Export(path string) error {
	if path == "" {
		return ErrEmptyExportPath
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return fmt.Errorf("confy: failed to create directory %s: %w", dir, err)
	}

	if err := c.v.WriteConfigAs(path); err != nil {
		return fmt.Errorf("confy: failed to write config to %s: %w", path, err)
	}

	return nil
}
