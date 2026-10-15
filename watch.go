package confy

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"reflect"
	"strings"
	"time"

	"github.com/spf13/viper"
)

func (c *Config) startWatchLocked() error {
	if !c.opts.watchable || c.watching {
		return nil
	}
	p, ok := c.provider.(WatchProvider)
	if !ok {
		return ErrWatchUnsupported
	}
	w, err := p.Watch(c.ctx)
	if err != nil {
		if w != nil {
			_ = w.Close()
		}
		return err
	}
	if w == nil {
		return ErrInvalidProvider
	}
	c.watcher = w
	c.watching = true
	go c.consume(w)
	return nil
}
func (c *Config) consume(w Watcher) {
	defer func() {
		_ = w.Close()
		c.mu.Lock()
		if c.watcher == w {
			c.watching = false
		}
		c.mu.Unlock()
	}()
	for {
		change, err := w.Next(c.ctx)
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
			case <-c.ctx.Done():
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
	if c.closed {
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
	if c.remote() && reflect.DeepEqual(settings, c.lastSettings) {
		c.mu.Unlock()
		return nil
	}
	target := reflect.ValueOf(c.watchBound)
	if !target.IsValid() || target.Kind() != reflect.Ptr || target.IsNil() {
		c.mu.Unlock()
		return ErrNilTarget
	}
	temporary := reflect.New(target.Elem().Type())
	err := c.decode(candidate, temporary.Interface(), c.watchDefaults)
	if err == nil && c.remote() {
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
	c.lastSettings = settings
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
	if c.closed {
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
	c.closeOnce.Do(func() {
		c.mu.Lock()
		c.closed = true
		w := c.watcher
		cancel := c.cancel
		c.mu.Unlock()
		if cancel != nil {
			cancel()
		}
		if w != nil {
			c.closeErr = w.Close()
		}
		if closer, ok := c.provider.(io.Closer); ok {
			c.closeErr = errors.Join(c.closeErr, closer.Close())
		}
	})
	return c.closeErr
}
