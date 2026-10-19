// Package provider defines the shared contracts implemented by configuration adapters.
package provider

import (
	"context"
	"reflect"
	"sync"

	"github.com/spf13/viper"
)

// Provider loads a complete source configuration. Watching providers update the
// same Viper before notifying their subscribers.
// A provider belongs to one Config; if it implements io.Closer, Config closes it.
type Provider interface {
	// Type identifies the implementation for explicit adapter dispatch.
	Type() ProviderType
	Load(context.Context) (*viper.Viper, error)
}

// SynchronizedProvider shares its Viper lock with Config. All provider writes
// must use this lock; callbacks must run after releasing it.
type SynchronizedProvider interface {
	Provider
	ConfigMutex() *sync.RWMutex
}

// WatchProvider optionally reports updates to the Viper returned by Load.
// Watch reports registration errors synchronously. Callbacks run asynchronously
// and serially, outside provider locks, and may call Config methods or Close.
// Context cancellation stops delivery; a nonterminal onError permits recovery.
type WatchProvider interface {
	SynchronizedProvider
	Watch(
		ctx context.Context,
		onChange func(Change),
		onError func(error),
	) (Watcher, error)
}

// Watcher stops delivery and releases resources when supported by the source.
// Close must be idempotent and safe to call from a callback.
type Watcher interface {
	Close() error
}

// Change identifies an update already applied to the shared Viper. Config must
// be the same instance returned by Load; callbacks may read it under ConfigMutex.
type Change struct {
	Config *viper.Viper
	Event  Event
}

// ProviderType identifies an adapter without importing its implementation.
type ProviderType string

const (
	ProviderEnv    ProviderType = "env"
	ProviderLocal  ProviderType = "local"
	ProviderNacos  ProviderType = "nacos"
	ProviderCustom ProviderType = "custom"
)

// IsNilProvider detects both nil interfaces and typed nil values before calling a Provider.
func IsNilProvider(p Provider) bool {
	if p == nil {
		return true
	}
	v := reflect.ValueOf(p)
	switch v.Kind() {
	case reflect.Ptr, reflect.Map, reflect.Func, reflect.Interface, reflect.Slice, reflect.Chan:
		return v.IsNil()
	}
	return false
}
