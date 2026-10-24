package engine

import "darkhold/internal/types"

// EventBus wraps a buffered channel for broadcasting events
type EventBus struct {
	C chan types.Event
}

func NewEventBus(bufferSize int) *EventBus {
	return &EventBus{
		C: make(chan types.Event, bufferSize),
	}
}

// Publish sends an event non-blockingly using select + default
func (eb *EventBus) Publish(event types.Event) {
	select {
	case eb.C <- event:
	default:
		// Drop event if buffer is full to prevent blocking the engine
	}
}