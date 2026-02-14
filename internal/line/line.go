package line

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

// Part represents an individual item flowing through a production line.
type Part struct {
	ID              string
	OrderID         string
	OperationID     string
	IsGood          bool
	CreatedAt       time.Time
	EnteredBufferAt time.Time
}

// NewPart creates a new part.
func NewPart(orderID, operationID string) *Part {
	return &Part{
		ID:        uuid.New().String(),
		OrderID:   orderID,
		OperationID: operationID,
		CreatedAt: time.Now(),
	}
}

// Buffer is a thread-safe FIFO queue between consecutive machines.
type Buffer struct {
	mu sync.RWMutex

	ID                  string
	LineName            string
	UpstreamMachineID   string
	DownstreamMachineID string
	MaxCapacity         int

	parts           []*Part
	totalThroughput int64
}

// NewBuffer creates a new buffer.
func NewBuffer(id, lineName, upstreamID, downstreamID string, capacity int) *Buffer {
	return &Buffer{
		ID:                  id,
		LineName:            lineName,
		UpstreamMachineID:   upstreamID,
		DownstreamMachineID: downstreamID,
		MaxCapacity:         capacity,
		parts:               make([]*Part, 0, capacity),
	}
}

// Add enqueues a part. Returns false if the buffer is full.
func (b *Buffer) Add(p *Part) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	if len(b.parts) >= b.MaxCapacity {
		return false
	}
	p.EnteredBufferAt = time.Now()
	b.parts = append(b.parts, p)
	return true
}

// Remove dequeues a part (FIFO). Returns nil if empty.
func (b *Buffer) Remove() *Part {
	b.mu.Lock()
	defer b.mu.Unlock()
	if len(b.parts) == 0 {
		return nil
	}
	p := b.parts[0]
	b.parts = b.parts[1:]
	b.totalThroughput++
	return p
}

// Peek returns the front part without removing it.
func (b *Buffer) Peek() *Part {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if len(b.parts) == 0 {
		return nil
	}
	return b.parts[0]
}

// IsFull returns true if the buffer is at capacity.
func (b *Buffer) IsFull() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.parts) >= b.MaxCapacity
}

// IsEmpty returns true if the buffer has no parts.
func (b *Buffer) IsEmpty() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.parts) == 0
}

// Count returns the number of parts in the buffer.
func (b *Buffer) Count() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.parts)
}

// Clear removes all parts from the buffer.
func (b *Buffer) Clear() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	cleared := len(b.parts)
	b.parts = b.parts[:0]
	return cleared
}

// TotalThroughput returns the total number of parts that have passed through.
func (b *Buffer) TotalThroughput() int64 {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.totalThroughput
}

// SetCapacity changes the buffer's max capacity at runtime.
func (b *Buffer) SetCapacity(cap int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.MaxCapacity = cap
}

// Line represents an active production line instance.
type Line struct {
	InstanceID   string
	TemplateName string
	Domain       string
	MachineIDs   []string // ordered by stage
	Buffers      map[string]*Buffer // keyed by buffer ID
	Recipes      []RecipeRef
}

// OrderedBuffers returns buffers in stage order (stage-1→2, stage-2→3, etc).
func (l *Line) OrderedBuffers() []*Buffer {
	result := make([]*Buffer, 0, len(l.Buffers))
	for i := 0; i < len(l.MachineIDs)-1; i++ {
		upID := l.MachineIDs[i]
		downID := l.MachineIDs[i+1]
		for _, b := range l.Buffers {
			if b.UpstreamMachineID == upID && b.DownstreamMachineID == downID {
				result = append(result, b)
				break
			}
		}
	}
	return result
}

// RecipeRef is a lightweight reference to a recipe.
type RecipeRef struct {
	ProductID   string
	Description string
}
