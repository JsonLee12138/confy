package confy

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/spf13/viper"
)

// watchState owns a single subscription and its active binding metadata.
type watchState struct {
	watcher      Watcher
	active       bool
	target       any
	defaults     bool
	lastSettings map[string]any
}

// lifecycleState synchronizes idempotent shutdown independently of callbacks.
type lifecycleState struct {
	ctx    context.Context
	cancel context.CancelFunc
	closed bool
	once   sync.Once
	err    error
}

func (c *Config) startWatchLocked() error {
	if !c.opts.watchable || c.watch.active {
		return nil
	}
	p, ok := c.provider.(WatchProvider)
	if !ok {
		return ErrWatchUnsupported
	}
	w, err := p.Watch(c.lifecycle.ctx)
	if err != nil {
		if w != nil {
			_ = w.Close()
		}
		return err
	}
	if w == nil {
		return ErrInvalidProvider
	}
	c.watch.watcher = w
	c.watch.active = true
	go c.consume(w)
	return nil
}
func (c *Config) consume(w Watcher) {
	defer func() {
		_ = w.Close()
		c.mu.Lock()
		if c.watch.watcher == w {
			c.watch.active = false
		}
		c.mu.Unlock()
	}()
	for {
		change, err := w.Next(c.lifecycle.ctx)
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) || errors.Is(err, io.EOF) {
			return
		}
		stage := "watch-next"
		if err == nil {
			stage = "candidate"
			err = c.commit(change)
		}
		if err != nil {
			c.notifyError(stage)
			// A broken third-party Watcher cannot turn repeated errors into a busy loop.
			select {
			case <-c.lifecycle.ctx.Done():
				return
			case <-time.After(25 * time.Millisecond):
			}
		}
	}
}
func (c *Config) compose(v *viper.Viper) {
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	if c.opts.envPrefix != "" {
		v.SetEnvPrefix(c.opts.envPrefix)
	}
	v.AutomaticEnv()
	applyEnvOverrides(v, c.opts.envPrefix)
	for key, value := range c.overrides {
		v.Set(key, value)
	}
}
func (c *Config) commit(change Change) error {
	c.mu.Lock()
	if c.lifecycle.closed {
		c.mu.Unlock()
		return nil
	}
	if change.Config == nil {
		c.mu.Unlock()
		return ErrInvalidProvider
	}
	candidate := change.Config
	c.compose(candidate)
	settings := candidate.AllSettings()
	if adapterFor(c.provider).deduplicate && reflect.DeepEqual(settings, c.watch.lastSettings) {
		c.mu.Unlock()
		return nil
	}
	target := reflect.ValueOf(c.watch.target)
	if !target.IsValid() || target.Kind() != reflect.Ptr || target.IsNil() {
		c.mu.Unlock()
		return ErrNilTarget
	}
	temporary := reflect.New(target.Elem().Type())
	err := c.decode(candidate, temporary.Interface(), c.watch.defaults)
	if err == nil && adapterFor(c.provider).validateUpdates {
		if validator, ok := temporary.Interface().(Validator); ok {
			if validator.Validate() != nil {
				err = ErrCandidate
			}
		}
	}
	if err != nil {
		c.mu.Unlock()
		return ErrCandidate
	}
	target.Elem().Set(temporary.Elem())
	c.v = candidate
	c.watch.lastSettings = settings
	fn := c.opts.onChange
	// Claim callback execution while holding the state lock. Close prevents all
	// subsequent claims; callbacks already claimed may finish, including Close.
	c.mu.Unlock()
	if fn != nil {
		fn(change.Event)
	}
	return nil
}
func (c *Config) notifyError(stage string) {
	c.mu.Lock()
	if c.lifecycle.closed {
		c.mu.Unlock()
		return
	}
	fn := c.opts.onError
	c.mu.Unlock()
	// Never expose provider/decoder/validator errors containing config contents.
	if fn != nil {
		fn(fmt.Errorf("%w: stage=%s", ErrCandidate, stage))
	} else {
		log.Printf("confy: configuration update failed: stage=%s", stage)
	}
}

// Close stops watching and releases owned provider resources. Get remains usable.
// It does not wait for callbacks, so calling Close from a callback is safe.
func (c *Config) Close() error {
	if c == nil {
		return nil
	}
	c.lifecycle.once.Do(func() {
		c.mu.Lock()
		c.lifecycle.closed = true
		w := c.watch.watcher
		cancel := c.lifecycle.cancel
		c.mu.Unlock()
		if cancel != nil {
			cancel()
		}
		if w != nil {
			c.lifecycle.err = w.Close()
		}
		if closer, ok := c.provider.(io.Closer); ok {
			c.lifecycle.err = errors.Join(c.lifecycle.err, closer.Close())
		}
	})
	return c.lifecycle.err
}
