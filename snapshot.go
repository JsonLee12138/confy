package confy

import "fmt"

// Snapshot captures the current config state for later restoration.
func (c *Config) Snapshot() (map[string]any, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	snapshot := make(map[string]any)
	if err := c.v.Unmarshal(&snapshot); err != nil {
		return nil, fmt.Errorf("confy: failed to create snapshot: %w", err)
	}

	c.snapshot = snapshot
	return snapshot, nil
}

// Restore restores config from a snapshot.
func (c *Config) Restore(snapshot map[string]any) error {
	if snapshot == nil {
		return ErrNilSnapshot
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	for k, v := range snapshot {
		c.v.Set(k, v)
	}

	c.snapshot = snapshot
	return nil
}
