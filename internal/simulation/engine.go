package simulation

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"machine-simulator-2/internal/event"
	"machine-simulator-2/internal/machine"
)

// Engine coordinates all machines and manages simulation-wide parameters.
type Engine struct {
	mu       sync.RWMutex
	machines map[string]machine.Machine
	bus      *event.Bus

	timeScale    float64
	randomFactor float64

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// New creates a new simulation engine.
func New(bus *event.Bus, timeScale, randomFactor float64) *Engine {
	return &Engine{
		machines:     make(map[string]machine.Machine),
		bus:          bus,
		timeScale:    timeScale,
		randomFactor: randomFactor,
	}
}

// RegisterMachine adds a machine to the engine.
func (e *Engine) RegisterMachine(m machine.Machine) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.machines[m.ID()] = m

	// Propagate current time scale and random factor
	setTimeScale(m, e.timeScale)
	setRandomFactor(m, e.randomFactor)

	// Wire state change callback via part complete
	m.SetOnPartComplete(func(machineID string, good bool) {
		if e.bus != nil {
			e.bus.Publish(event.Event{
				Type:      event.MachineCycleComplete,
				MachineID: machineID,
				Data:      good,
			})
		}
	})
}

// GetMachine returns a machine by ID.
func (e *Engine) GetMachine(id string) machine.Machine {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.machines[id]
}

// GetAllMachines returns all registered machines, sorted by ID.
func (e *Engine) GetAllMachines() []machine.Machine {
	e.mu.RLock()
	defer e.mu.RUnlock()
	result := make([]machine.Machine, 0, len(e.machines))
	for _, m := range e.machines {
		result = append(result, m)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].ID() < result[j].ID()
	})
	return result
}

// Start launches all registered machines.
func (e *Engine) Start(ctx context.Context) {
	e.mu.Lock()
	e.ctx, e.cancel = context.WithCancel(ctx)
	e.mu.Unlock()

	e.mu.RLock()
	for _, m := range e.machines {
		e.wg.Add(1)
		go func(m machine.Machine) {
			defer e.wg.Done()
			m.Start(e.ctx)
		}(m)
	}
	e.mu.RUnlock()
}

// Stop cancels all machine goroutines and waits for them to finish.
func (e *Engine) Stop() {
	e.mu.RLock()
	cancel := e.cancel
	e.mu.RUnlock()

	if cancel != nil {
		cancel()
	}
	e.wg.Wait()
}

// SendCommand routes a command to a specific machine.
func (e *Engine) SendCommand(machineID string, cmd machine.Command) error {
	m := e.GetMachine(machineID)
	if m == nil {
		return fmt.Errorf("machine %q not found", machineID)
	}
	return m.SendCommand(cmd)
}

// SetTimeScale updates the time scale on all machines.
func (e *Engine) SetTimeScale(scale float64) {
	e.mu.Lock()
	e.timeScale = scale
	machines := make([]machine.Machine, 0, len(e.machines))
	for _, m := range e.machines {
		machines = append(machines, m)
	}
	e.mu.Unlock()

	for _, m := range machines {
		setTimeScale(m, scale)
	}
}

// GetTimeScale returns the current time scale.
func (e *Engine) GetTimeScale() float64 {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.timeScale
}

// SetRandomFactor updates the random factor on all machines.
func (e *Engine) SetRandomFactor(factor float64) {
	e.mu.Lock()
	e.randomFactor = factor
	machines := make([]machine.Machine, 0, len(e.machines))
	for _, m := range e.machines {
		machines = append(machines, m)
	}
	e.mu.Unlock()

	for _, m := range machines {
		setRandomFactor(m, factor)
	}
}

// GetRandomFactor returns the current random factor.
func (e *Engine) GetRandomFactor() float64 {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.randomFactor
}

// MachineCount returns the number of registered machines.
func (e *Engine) MachineCount() int {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return len(e.machines)
}

// baseMachineAccessor is used for type assertion (unused placeholder).
type baseMachineAccessor interface {
	SetTimeScale(float64)
	SetRandomFactor(float64)
}

func setTimeScale(m machine.Machine, scale float64) {
	if s, ok := m.(interface{ SetTimeScale(float64) }); ok {
		s.SetTimeScale(scale)
	}
}

func setRandomFactor(m machine.Machine, factor float64) {
	if s, ok := m.(interface{ SetRandomFactor(float64) }); ok {
		s.SetRandomFactor(factor)
	}
}
