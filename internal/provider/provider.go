package provider

import "context"

// Provider defines the interface for remote configuration sources.
// This is a v2 placeholder — not used in v1.
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
	EventPut    EventType = iota
	EventDelete
)
