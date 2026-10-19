package confy

import "github.com/JsonLee12138/confy/provider"

// Validator is an interface that config structs can implement
// to perform custom validation.
type Validator interface {
	Validate() error
}

// ConfigInterface defines the public contract for a Config instance.
type ConfigInterface interface {
	Bind(instance any) error

	// BindWithDefaults is a compatibility alias for Bind.
	//
	// Deprecated: Use Bind instead; it applies default tags automatically.
	BindWithDefaults(instance any) error
	Validate() error
	ValidateType(instance any) error
	Export(path string) error
	Snapshot() (map[string]any, error)
	Restore() error
	Get(key string) any
	Set(key string, value any)
}

// Aliases keep existing option and event signatures source-compatible.
type FileType = provider.FileType

const (
	YAML = provider.YAML
	JSON = provider.JSON
	TOML = provider.TOML
)

type Event = provider.Event
type EventOp = provider.EventOp

const (
	EventCreate = provider.EventCreate
	EventWrite  = provider.EventWrite
	EventRemove = provider.EventRemove
	EventRename = provider.EventRename
	EventChmod  = provider.EventChmod
)

// Compatibility aliases; new adapters should import package provider directly.
type Provider = provider.Provider
type WatchProvider = provider.WatchProvider
type Watcher = provider.Watcher
type Change = provider.Change
type ProviderType = provider.ProviderType

const (
	ProviderEnv    = provider.ProviderEnv
	ProviderLocal  = provider.ProviderLocal
	ProviderNacos  = provider.ProviderNacos
	ProviderCustom = provider.ProviderCustom
)
