package event

import "sync"

// Event type constants.
const (
	MachineStateChange = "machine.state_change"
	MachineCycleComplete = "machine.cycle_complete"
	LinePartMoved      = "line.part_moved"
	OrderCreated       = "order.created"
	OrderReleased      = "order.released"
	OrderProgress      = "order.progress"
	OrderClosed        = "order.closed"
	BufferFull         = "buffer.full"
	BufferEmpty        = "buffer.empty"
	LineDeadlockRecovered = "line.deadlock_recovered"
)

// Event represents a typed event published on the bus.
type Event struct {
	Type      string
	MachineID string
	LineID    string
	Data      interface{}
}

// Handler is a callback invoked when a matching event is published.
type Handler func(Event)

// Bus is a thread-safe publish/subscribe event bus.
type Bus struct {
	mu       sync.RWMutex
	handlers map[string][]Handler
}

// New creates a new event bus.
func New() *Bus {
	return &Bus{
		handlers: make(map[string][]Handler),
	}
}

// Subscribe registers a handler for the given event type.
func (b *Bus) Subscribe(eventType string, handler Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[eventType] = append(b.handlers[eventType], handler)
}

// Publish sends an event to all handlers registered for its type.
// Handlers are called synchronously in registration order.
func (b *Bus) Publish(e Event) {
	b.mu.RLock()
	handlers := make([]Handler, len(b.handlers[e.Type]))
	copy(handlers, b.handlers[e.Type])
	b.mu.RUnlock()

	for _, h := range handlers {
		h(e)
	}
}
