package confy

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

// FileType represents supported configuration file formats.
type FileType string

const (
	YAML FileType = "yaml"
	JSON FileType = "json"
	TOML FileType = "toml"
)

// Event represents a configuration file change event.
type Event struct {
	Name string
	Op   EventOp
}

type EventOp uint32

const (
	EventCreate EventOp = 1 << iota
	EventWrite
	EventRemove
	EventRename
	EventChmod
)
