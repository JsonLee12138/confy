package confy

import (
	"context"
	"github.com/spf13/viper"
	"reflect"
)

// Provider transfers ownership of a fresh configuration to Config on each load.
// A provider belongs to one Config; if it implements io.Closer, Config closes it.
type Provider interface {
	Load(context.Context) (*viper.Viper, error)
}

// WatchProvider optionally supplies complete replacement configurations.
type WatchProvider interface {
	Provider
	Watch(context.Context) (Watcher, error)
}

// Watcher delivers changes serially. Close must be idempotent and unblock Next.
// Nonterminal errors reject one update; EOF and context errors end consumption.
type Watcher interface {
	Next(context.Context) (Change, error)
	Close() error
}

// Change contains an independently owned candidate and a safe source identifier.
type Change struct {
	Config *viper.Viper
	Event  Event
}

func isNilProvider(p Provider) bool {
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
