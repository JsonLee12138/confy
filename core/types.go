package core

import "github.com/JsonLee12138/confy/provider"

// FileType represents supported configuration file formats.
type FileType = provider.FileType

const (
	YAML = provider.YAML
	JSON = provider.JSON
	TOML = provider.TOML
)

// Event represents a configuration file change event.
type Event = provider.Event

type EventOp = provider.EventOp

const (
	EventCreate = provider.EventCreate
	EventWrite  = provider.EventWrite
	EventRemove = provider.EventRemove
	EventRename = provider.EventRename
	EventChmod  = provider.EventChmod
)
