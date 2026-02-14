package event

import (
	"sync"
	"sync/atomic"
	"testing"
)

func TestSubscribeAndPublish(t *testing.T) {
	bus := New()
	var received Event

	bus.Subscribe(MachineStateChange, func(e Event) {
		received = e
	})

	bus.Publish(Event{
		Type:      MachineStateChange,
		MachineID: "m1",
		Data:      "idle->running",
	})

	if received.MachineID != "m1" {
		t.Errorf("expected MachineID m1, got %s", received.MachineID)
	}
	if received.Data != "idle->running" {
		t.Errorf("unexpected data: %v", received.Data)
	}
}

func TestFanOut(t *testing.T) {
	bus := New()
	var count int

	bus.Subscribe(OrderCreated, func(e Event) { count++ })
	bus.Subscribe(OrderCreated, func(e Event) { count++ })
	bus.Subscribe(OrderCreated, func(e Event) { count++ })

	bus.Publish(Event{Type: OrderCreated})

	if count != 3 {
		t.Errorf("expected 3 handlers called, got %d", count)
	}
}

func TestNoHandlerNoPanic(t *testing.T) {
	bus := New()
	// Should not panic when publishing with no subscribers.
	bus.Publish(Event{Type: BufferFull})
}

func TestDifferentEventTypes(t *testing.T) {
	bus := New()
	var machineCount, orderCount int

	bus.Subscribe(MachineStateChange, func(e Event) { machineCount++ })
	bus.Subscribe(OrderCreated, func(e Event) { orderCount++ })

	bus.Publish(Event{Type: MachineStateChange})
	bus.Publish(Event{Type: MachineStateChange})
	bus.Publish(Event{Type: OrderCreated})

	if machineCount != 2 {
		t.Errorf("expected machineCount 2, got %d", machineCount)
	}
	if orderCount != 1 {
		t.Errorf("expected orderCount 1, got %d", orderCount)
	}
}

func TestConcurrentPublishSafety(t *testing.T) {
	bus := New()
	var count atomic.Int64

	bus.Subscribe(MachineCycleComplete, func(e Event) {
		count.Add(1)
	})

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			bus.Publish(Event{Type: MachineCycleComplete})
		}()
	}
	wg.Wait()

	if count.Load() != 100 {
		t.Errorf("expected 100, got %d", count.Load())
	}
}

func TestConcurrentSubscribeAndPublish(t *testing.T) {
	bus := New()
	var count atomic.Int64

	var wg sync.WaitGroup
	// Concurrent subscribers
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			bus.Subscribe(LinePartMoved, func(e Event) {
				count.Add(1)
			})
		}()
	}
	wg.Wait()

	// Publish after all subscriptions
	bus.Publish(Event{Type: LinePartMoved})

	if count.Load() != 10 {
		t.Errorf("expected 10, got %d", count.Load())
	}
}
