package confy

import (
	"fmt"

	"github.com/creasty/defaults"
	"github.com/fsnotify/fsnotify"
)

// Bind unmarshals the config into the target struct.
// If watching is enabled, the target will be automatically updated on config changes.
func (c *Config) Bind(instance any) error {
	if c == nil || c.v == nil {
		return ErrNilConfig
	}
	if instance == nil {
		return ErrNilTarget
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.v.Unmarshal(&instance); err != nil {
		return fmt.Errorf("confy: failed to unmarshal config: %w", err)
	}

	c.bound = instance

	if c.opts.watchable {
		c.watchOnce.Do(func() {
			c.v.WatchConfig()
			c.v.OnConfigChange(func(e fsnotify.Event) {
				c.mu.Lock()
				defer c.mu.Unlock()

				if err := c.v.Unmarshal(&instance); err != nil {
					fmt.Printf("confy: watch rebind error: %v\n", err)
					return
				}

				if c.opts.onChange != nil {
					c.opts.onChange(Event{
						Name: e.Name,
						Op:   EventOp(e.Op),
					})
				}
			})
		})
	}

	return nil
}

// BindWithDefaults applies struct default values before and after binding.
func (c *Config) BindWithDefaults(instance any) error {
	if err := defaults.Set(instance); err != nil {
		return fmt.Errorf("confy: failed to set defaults: %w", err)
	}

	if err := c.Bind(instance); err != nil {
		return err
	}

	if err := defaults.Set(instance); err != nil {
		return fmt.Errorf("confy: failed to set defaults after bind: %w", err)
	}

	return nil
}
