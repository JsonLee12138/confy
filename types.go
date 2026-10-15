package confy

import "github.com/JsonLee12138/confy/core"

// Validator is an interface that config structs can implement
// to perform custom validation.
type Validator interface {
	Validate() error
}

// ConfigInterface defines the public contract for a Config instance.
type ConfigInterface interface {
	Bind(instance any) error
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
type FileType = core.FileType

const (
	YAML = core.YAML
	JSON = core.JSON
	TOML = core.TOML
)

type Event = core.Event
type EventOp = core.EventOp

const (
	EventCreate = core.EventCreate
	EventWrite  = core.EventWrite
	EventRemove = core.EventRemove
	EventRename = core.EventRename
	EventChmod  = core.EventChmod
)
