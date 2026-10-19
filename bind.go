package confy

import (
	"fmt"
	"log"
	"reflect"

	"github.com/JsonLee12138/confy/core"
	"github.com/JsonLee12138/confy/provider"
	"github.com/creasty/defaults"
)

// Bind initializes a fresh struct with defaults once, then decodes configuration
// over it, preserving explicit zero values. Success replaces the target; decode
// failure leaves it unchanged. The optional watcher uses the same pipeline.
// Applications must synchronize direct reads of a target that is auto-updated.
func (c *Config) Bind(instance any) error {
	if c == nil || c.mu == nil {
		return ErrNilConfig
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return ErrClosed
	}
	if err := c.bind(instance); err != nil {
		return err
	}
	c.bound = instance
	if !c.opts.watchable || c.watcher != nil {
		return nil
	}
	source, ok := c.provider.(provider.WatchProvider)
	if !ok {
		return ErrWatchUnsupported
	}
	watcher, err := source.Watch(c.ctx, func(change provider.Change) {
		c.mu.Lock()
		if c.closed {
			c.mu.Unlock()
			return
		}
		err := c.bind(instance)
		fn := c.opts.onChange
		c.mu.Unlock()
		if err != nil {
			c.bindingError("bind")
			return
		}
		if fn != nil {
			fn(change.Event)
		}
	}, func(error) { c.bindingError("watch") })
	if err != nil {
		if watcher != nil {
			_ = watcher.Close()
		}
		return err
	}
	if watcher == nil {
		return ErrInvalidProvider
	}
	c.watcher = watcher
	return nil
}

// BindWithDefaults is a compatibility alias for Bind.
//
// Deprecated: Use [Config.Bind] instead; it applies default tags automatically.
func (c *Config) BindWithDefaults(instance any) error { return c.Bind(instance) }

// bind applies the same Viper binding rules to every source and every update.
func (c *Config) bind(instance any) error {
	target := reflect.ValueOf(instance)
	// defaults.Set panics for nil pointers; other target errors are left to it.
	if target.Kind() != reflect.Ptr || target.IsNil() {
		return ErrMustBeStruct
	}
	decoded := reflect.New(target.Elem().Type())
	if err := defaults.Set(decoded.Interface()); err != nil {
		return err
	}
	v, err := c.bindingViper()
	if err != nil {
		return err
	}
	if err := v.Unmarshal(decoded.Interface(), core.DecodeOption()); err != nil {
		return err
	}
	target.Elem().Set(decoded.Elem())
	return nil
}

// bindingError reports a phase-only update error without exposing configuration values.
// It suppresses delivery after Close and invokes the error callback outside the lock.
func (c *Config) bindingError(stage string) {
	c.mu.RLock()
	fn, closed := c.opts.onError, c.closed
	c.mu.RUnlock()
	if closed {
		return
	}
	if fn != nil {
		fn(fmt.Errorf("%w: stage=%s", ErrCandidate, stage))
	} else {
		log.Printf("confy: configuration update failed: stage=%s", stage)
	}
}
