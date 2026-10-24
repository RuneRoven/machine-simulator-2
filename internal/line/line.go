package line

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Part represents an individual item flowing through a production line.
type Part struct {
	ID              string
	JobID           string
	OrderID         string
	OperationID     string
	ProductID       string
	IsGood          bool
	CreatedAt       time.Time
	EnteredBufferAt time.Time
}

// NewPart creates a new part.
func NewPart(orderID, operationID string) *Part {
	return &Part{
		ID:          uuid.New().String(),
		OrderID:     orderID,
		OperationID: operationID,
		CreatedAt:   time.Now(),
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
	totalIn         int64
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
	b.totalIn++
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

// Capacity returns the buffer's max capacity.
func (b *Buffer) Capacity() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.MaxCapacity
}

// forEach calls fn for every part in the buffer, front first.
func (b *Buffer) forEach(fn func(p *Part)) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for _, p := range b.parts {
		fn(p)
	}
}

// removeIf removes the parts matching match and returns how many it removed.
// Removed parts do not count as throughput.
func (b *Buffer) removeIf(match func(p *Part) bool) int {
	b.mu.Lock()
	defer b.mu.Unlock()
	kept := b.parts[:0]
	removed := 0
	for _, p := range b.parts {
		if match(p) {
			removed++
			continue
		}
		kept = append(kept, p)
	}
	for i := len(kept); i < len(b.parts); i++ {
		b.parts[i] = nil
	}
	b.parts = kept
	return removed
}

// TotalIn returns the number of parts that have entered the buffer.
func (b *Buffer) TotalIn() int64 {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.totalIn
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
	MachineIDs   []string           // ordered by stage
	Stages       []int              // stage number per entry of MachineIDs
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

// ValidateProduct returns ErrUnknownProduct when the line has a recipe list and
// productID is not on it.
func (l *Line) ValidateProduct(productID string) error {
	if len(l.Recipes) == 0 {
		return nil
	}
	valid := make([]string, 0, len(l.Recipes))
	for _, r := range l.Recipes {
		if r.ProductID == productID {
			return nil
		}
		valid = append(valid, r.ProductID)
	}
	return fmt.Errorf("%w %q for line %s, valid: %v", ErrUnknownProduct, productID, l.InstanceID, valid)
}
