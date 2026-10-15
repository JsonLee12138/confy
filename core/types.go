package core

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
