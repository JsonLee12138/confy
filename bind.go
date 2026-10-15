package confy

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/creasty/defaults"
	"github.com/spf13/viper"
)

// Bind decodes configuration and starts the optional watcher after success.
// Applications must synchronize direct reads of a target that is auto-updated.
func (c *Config) Bind(instance any) error { return c.bind(instance, false) }

// BindWithDefaults applies defaults before and after decoding, including reloads.
func (c *Config) BindWithDefaults(instance any) error { return c.bind(instance, true) }
func (c *Config) bind(instance any, withDefaults bool) error {
	if c == nil {
		return ErrNilConfig
	}
	if instance == nil {
		return ErrNilTarget
	}
	target := reflect.ValueOf(instance)
	if target.Kind() != reflect.Ptr {
		return ErrMustBeStruct
	}
	if target.IsNil() {
		return ErrNilTarget
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return ErrClosed
	}
	if c.v == nil {
		return ErrNilConfig
	}
	if c.remote() && c.bound != nil && c.bound != instance {
		return ErrBoundTarget
	}
	settings := c.v.AllSettings()
	if err := c.decode(c.v, instance, withDefaults); err != nil {
		return err
	}
	c.bound = instance
	if c.lastSettings == nil {
		c.lastSettings = settings
	}
	// Historical local watching updates the first target; Validate uses the last.
	if c.watchBound == nil || c.watchBound == instance {
		c.watchBound = instance
		c.watchDefaults = withDefaults
	}
	return c.startWatchLocked()
}
func (c *Config) remote() bool { _, local := c.provider.(*LocalProvider); return !local }
func (c *Config) decode(v *viper.Viper, target any, withDefaults bool) error {
	if withDefaults {
		if err := defaults.Set(target); err != nil {
			return fmt.Errorf("confy: failed to set defaults: %w", err)
		}
	}
	for _, key := range v.AllKeys() {
		value := v.GetString(key)
		if c.remote() && strings.HasPrefix(value, encryptedPrefix) && (c.opts.encAlgo != "aes-gcm" || len(c.opts.encKey) == 0 || !isEncryptedValue(value)) {
			return ErrCandidate
		}
		if c.opts.encAlgo == "aes-gcm" && len(c.opts.encKey) > 0 && isEncryptedValue(value) {
			plain, err := decryptConfigValue(value, c.opts.encKey)
			if err != nil {
				if c.remote() {
					return ErrCandidate
				}
				return fmt.Errorf("confy: failed to decrypt key %s: %w", key, err)
			}
			v.Set(key, plain)
		}
	}
	if err := v.Unmarshal(target); err != nil {
		if c.remote() {
			return ErrCandidate
		}
		return fmt.Errorf("confy: failed to unmarshal config: %w", err)
	}
	if withDefaults {
		if err := defaults.Set(target); err != nil {
			return fmt.Errorf("confy: failed to set defaults after bind: %w", err)
		}
	}
	return nil
}
