// Package core defines the shared contracts implemented by configuration adapters.
package core

import (
	"context"

	"github.com/spf13/viper"
)

// Provider transfers ownership of a fresh configuration to Config on each load.
// A provider belongs to one Config; if it implements io.Closer, Config closes it.
type Provider interface {
	// Type identifies the implementation for explicit adapter dispatch.
	Type() ProviderType
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

// ProviderType identifies an adapter without importing its implementation.
type ProviderType string

const (
	ProviderLocal  ProviderType = "local"
	ProviderNacos  ProviderType = "nacos"
	ProviderCustom ProviderType = "custom"
)
