package confy

import (
	"fmt"
)

// Snapshot decodes effective settings into a new map for later Restore.
// ENV values can appear as plaintext; source ciphertext is not decrypted here.
func (c *Config) Snapshot() (map[string]any, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	snapshot := make(map[string]any)
	if err := c.v.Unmarshal(&snapshot); err != nil {
		return nil, fmt.Errorf("confy: failed to create snapshot: %w", err)
	}

	return snapshot, nil
}

// Restore applies snapshot entries as Viper overrides without replacing the source.
// It does not remove keys absent from the snapshot or rebind targets. A nil map
// returns ErrNilSnapshot, and a closed Config returns ErrClosed.
func (c *Config) Restore(snapshot map[string]any) error {
	if snapshot == nil {
		return ErrNilSnapshot
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return ErrClosed
	}
	for k, v := range snapshot {
		c.v.Set(k, v)
	}

	return nil
}
