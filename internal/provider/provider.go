package provider

import "context"

// Provider defines the interface for remote configuration sources.
// Deprecated: unused legacy placeholder. Active adapters implement the public
// confy.Provider and confy.WatchProvider contracts instead.
type Provider interface {
	Load(ctx context.Context) (map[string]any, error)
	Watch(ctx context.Context, onChange func(Event)) error
	Close() error
}

type Event struct {
	Key   string
	Value any
	Type  EventType
}

type EventType int

const (
	EventPut EventType = iota
	EventDelete
)
