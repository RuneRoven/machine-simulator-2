package machine

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"machine-simulator-2/internal/config"
	"machine-simulator-2/internal/event"
)

// State represents a machine's operational state.
type State string

const (
	StateIdle        State = "IDLE"
	StateRunning     State = "RUNNING"
	StateStopped     State = "STOPPED"
	StateFault       State = "FAULT"
	StateMaintenance State = "MAINTENANCE"
)

// Command represents a machine control command.
type Command string

const (
	CommandStart       Command = "START"
	CommandStop        Command = "STOP"
	CommandReset       Command = "RESET"
	CommandFault       Command = "FAULT"
	CommandMaintenance Command = "MAINTENANCE"
)

// ValidTransitions defines allowed state transitions.
var ValidTransitions = map[State][]State{
	StateIdle:        {StateRunning},
	StateRunning:     {StateIdle, StateStopped, StateFault, StateMaintenance},
	StateStopped:     {StateIdle},
	StateFault:       {StateIdle},
	StateMaintenance: {StateIdle},
}

// Machine is the interface that all machine types must implement.
type Machine interface {
	// Identity
	ID() string
	Type() string
	Role() string
	LineInstanceID() string

	// Lifecycle
	Start(ctx context.Context)
	Stop()
	SendCommand(cmd Command) error

	// State
	GetState() State
	GetBaseData() BaseData

	// Tags
	GetTags() map[string]interface{}
	GetTagDefinitions() []TagDefinition

	// Production
	AssignOrder(orderID, operationID, productID string)
	ClearOrder()
	GetCurrentOrder() (orderID, operationID string)

	// Runtime config
	ApplyRecipeOverrides(overrides RecipeOverrides)
	UpdateConfig(settings MachineSettings)
	GetSettings() MachineSettings

	// Line integration
	SetOnPartComplete(fn func(machineID string, good bool))
	CanAcceptPart() bool
	IsBlockedByBuffer() bool
	SetBlockedByBuffer(blocked bool)
	SetStarvedByBuffer(starved bool)
	IsStarvedByBuffer() bool
}

// BaseData holds the current state snapshot of a machine.
type BaseData struct {
	ID               string
	Type             string
	Role             string
	LineInstanceID   string
	State            State
	CycleCount       int64
	GoodCount        int64
	ScrapCount       int64
	CurrentOrderID   string
	CurrentOpID      string
	CurrentProductID string
	RunTimeSec       float64
	DownTimeSec      float64
}

// MachineSettings holds runtime-adjustable machine parameters.
type MachineSettings struct {
	CycleTimeMS          int     `json:"cycle_time_ms"`
	ScrapRate            float64 `json:"scrap_rate"`
	BreakdownProbability float64 `json:"breakdown_probability"`
	AutoRestoreTimeMS    int     `json:"auto_restore_time_ms"`
}

// RecipeOverrides holds per-stage recipe parameter overrides.
type RecipeOverrides struct {
	CycleTimeMS          *int
	ScrapRate            *float64
	BreakdownProbability *float64
}

// BaseMachine provides a default implementation of the Machine interface.
// Machine types embed this and override GetTags, GetTagDefinitions, and Start.
type BaseMachine struct {
	mu sync.RWMutex

	id             string
	machineType    string
	role           string
	lineInstanceID string

	state       State
	cycleCount  int64
	goodCount   int64
	scrapCount  int64

	// Timing
	cycleTimeMS          int
	scrapRate            float64
	breakdownProbability float64
	autoRestoreTimeMS    int
	tickRateMS           int
	timeScale            float64
	randomFactor         float64

	// Saved defaults for recipe override revert
	savedCycleTimeMS          int
	savedScrapRate            float64
	savedBreakdownProbability float64
	hasRecipeOverride         bool

	// Production tracking
	currentOrderID   string
	currentOpID      string
	currentProductID string
	cycleElapsedMS   float64

	// Runtime tracking
	runStart  time.Time
	runTimeNS int64
	downStart time.Time
	downTimeNS int64

	// Buffer integration
	blockedByBuffer  bool
	starvedByBuffer  bool

	// Fault timing
	faultStart time.Time

	// Callbacks
	onPartComplete func(machineID string, good bool)

	// Event bus for publishing state changes
	eventBus *event.Bus

	// Command channel
	commandChan chan Command
	cancel      context.CancelFunc
	rng         *rand.Rand
}

// NewBaseMachine creates a new BaseMachine from config.
func NewBaseMachine(cfg config.MachineInstanceConfig) *BaseMachine {
	return &BaseMachine{
		id:                   cfg.ID,
		machineType:          cfg.Type,
		role:                 cfg.Role,
		lineInstanceID:       cfg.LineInstanceID,
		state:                StateIdle,
		cycleTimeMS:          cfg.CycleTimeMS,
		scrapRate:            cfg.ScrapRate,
		breakdownProbability: cfg.BreakdownProbability,
		autoRestoreTimeMS:    cfg.AutoRestoreTimeMS,
		tickRateMS:           100,
		timeScale:            1.0,
		randomFactor:         0.1,
		commandChan:          make(chan Command, 10),
		rng:                  rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (m *BaseMachine) ID() string             { m.mu.RLock(); defer m.mu.RUnlock(); return m.id }
func (m *BaseMachine) Type() string            { m.mu.RLock(); defer m.mu.RUnlock(); return m.machineType }
func (m *BaseMachine) Role() string            { m.mu.RLock(); defer m.mu.RUnlock(); return m.role }
func (m *BaseMachine) LineInstanceID() string   { m.mu.RLock(); defer m.mu.RUnlock(); return m.lineInstanceID }

func (m *BaseMachine) GetState() State {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.state
}

func (m *BaseMachine) GetBaseData() BaseData {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return BaseData{
		ID:               m.id,
		Type:             m.machineType,
		Role:             m.role,
		LineInstanceID:   m.lineInstanceID,
		State:            m.state,
		CycleCount:       m.cycleCount,
		GoodCount:        m.goodCount,
		ScrapCount:       m.scrapCount,
		CurrentOrderID:   m.currentOrderID,
		CurrentOpID:      m.currentOpID,
		CurrentProductID: m.currentProductID,
		RunTimeSec:       float64(m.runTimeNS) / float64(time.Second),
		DownTimeSec:      float64(m.downTimeNS) / float64(time.Second),
	}
}

func (m *BaseMachine) SendCommand(cmd Command) error {
	select {
	case m.commandChan <- cmd:
		return nil
	default:
		return fmt.Errorf("command channel full for machine %s", m.id)
	}
}

func (m *BaseMachine) AssignOrder(orderID, operationID, productID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.currentOrderID = orderID
	m.currentOpID = operationID
	m.currentProductID = productID
}

func (m *BaseMachine) ClearOrder() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.currentOrderID = ""
	m.currentOpID = ""
	m.currentProductID = ""
	if m.hasRecipeOverride {
		m.cycleTimeMS = m.savedCycleTimeMS
		m.scrapRate = m.savedScrapRate
		m.breakdownProbability = m.savedBreakdownProbability
		m.hasRecipeOverride = false
	}
}

func (m *BaseMachine) GetCurrentOrder() (orderID, operationID string) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.currentOrderID, m.currentOpID
}

func (m *BaseMachine) ApplyRecipeOverrides(overrides RecipeOverrides) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.hasRecipeOverride {
		m.savedCycleTimeMS = m.cycleTimeMS
		m.savedScrapRate = m.scrapRate
		m.savedBreakdownProbability = m.breakdownProbability
		m.hasRecipeOverride = true
	}
	if overrides.CycleTimeMS != nil {
		m.cycleTimeMS = *overrides.CycleTimeMS
	}
	if overrides.ScrapRate != nil {
		m.scrapRate = *overrides.ScrapRate
	}
	if overrides.BreakdownProbability != nil {
		m.breakdownProbability = *overrides.BreakdownProbability
	}
}

func (m *BaseMachine) UpdateConfig(settings MachineSettings) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if settings.CycleTimeMS > 0 {
		m.cycleTimeMS = settings.CycleTimeMS
	}
	if settings.ScrapRate >= 0 {
		m.scrapRate = settings.ScrapRate
	}
	if settings.BreakdownProbability >= 0 {
		m.breakdownProbability = settings.BreakdownProbability
	}
	if settings.AutoRestoreTimeMS > 0 {
		m.autoRestoreTimeMS = settings.AutoRestoreTimeMS
	}
}

func (m *BaseMachine) GetSettings() MachineSettings {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return MachineSettings{
		CycleTimeMS:          m.cycleTimeMS,
		ScrapRate:            m.scrapRate,
		BreakdownProbability: m.breakdownProbability,
		AutoRestoreTimeMS:    m.autoRestoreTimeMS,
	}
}

func (m *BaseMachine) SetOnPartComplete(fn func(machineID string, good bool)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onPartComplete = fn
}

func (m *BaseMachine) CanAcceptPart() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.state == StateIdle && m.currentOrderID != ""
}

func (m *BaseMachine) IsBlockedByBuffer() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.blockedByBuffer
}

func (m *BaseMachine) SetBlockedByBuffer(blocked bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.blockedByBuffer = blocked
}

func (m *BaseMachine) SetStarvedByBuffer(starved bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.starvedByBuffer = starved
}

func (m *BaseMachine) IsStarvedByBuffer() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.starvedByBuffer
}

// SetEventBus sets the event bus for publishing state change events.
func (m *BaseMachine) SetEventBus(bus *event.Bus) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.eventBus = bus
}

// SetTimeScale sets the time acceleration factor.
func (m *BaseMachine) SetTimeScale(scale float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.timeScale = scale
}

// SetRandomFactor sets the random variance factor.
func (m *BaseMachine) SetRandomFactor(factor float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.randomFactor = factor
}

// Start runs the machine's tick loop. Override in subtypes for custom tag updates.
func (m *BaseMachine) Start(ctx context.Context) {
	ctx, m.cancel = context.WithCancel(ctx)
	ticker := time.NewTicker(time.Duration(m.tickRateMS) * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case cmd := <-m.commandChan:
			m.processCommand(cmd)
		case <-ticker.C:
			m.tick()
		}
	}
}

// Stop cancels the machine's context.
func (m *BaseMachine) Stop() {
	if m.cancel != nil {
		m.cancel()
	}
}

// GetTags returns base tags. Subtypes should override this.
func (m *BaseMachine) GetTags() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return map[string]interface{}{
		"state":       string(m.state),
		"cycle_count": m.cycleCount,
		"good_count":  m.goodCount,
		"scrap_count": m.scrapCount,
	}
}

// GetTagDefinitions returns base tag definitions. Subtypes should override this.
func (m *BaseMachine) GetTagDefinitions() []TagDefinition {
	return []TagDefinition{
		{Name: "state", DataType: "string", Description: "Machine state"},
		{Name: "cycle_count", DataType: "int64", Description: "Total cycles completed"},
		{Name: "good_count", DataType: "int64", Description: "Good parts produced"},
		{Name: "scrap_count", DataType: "int64", Description: "Scrap parts produced"},
	}
}

func (m *BaseMachine) processCommand(cmd Command) {
	m.mu.Lock()
	defer m.mu.Unlock()

	switch cmd {
	case CommandStart:
		if m.canTransition(StateRunning) {
			m.setState(StateRunning)
		}
	case CommandStop:
		if m.canTransition(StateStopped) {
			m.setState(StateStopped)
		}
	case CommandReset:
		if m.state == StateFault || m.state == StateStopped || m.state == StateMaintenance {
			m.setState(StateIdle)
		}
	case CommandFault:
		if m.canTransition(StateFault) {
			m.setState(StateFault)
		}
	case CommandMaintenance:
		if m.canTransition(StateMaintenance) {
			m.setState(StateMaintenance)
		}
	}
}

func (m *BaseMachine) tick() {
	m.mu.Lock()
	defer m.mu.Unlock()

	switch m.state {
	case StateRunning:
		m.tickRunning()
	case StateFault:
		m.tickFault()
	}
}

func (m *BaseMachine) tickRunning() {
	if m.blockedByBuffer || m.starvedByBuffer {
		return
	}

	effectiveCycleTime := m.effectiveCycleTimeMS()
	m.cycleElapsedMS += float64(m.tickRateMS)

	if m.cycleElapsedMS >= effectiveCycleTime {
		m.cycleElapsedMS = 0
		m.completeCycle()
	}
}

func (m *BaseMachine) completeCycle() {
	m.cycleCount++

	good := m.rng.Float64() >= m.scrapRate
	if good {
		m.goodCount++
	} else {
		m.scrapCount++
	}

	// Fire callback (without lock)
	if m.onPartComplete != nil {
		fn := m.onPartComplete
		id := m.id
		m.mu.Unlock()
		fn(id, good)
		m.mu.Lock()
	}

	// Check for random breakdown
	if m.breakdownProbability > 0 && m.rng.Float64() < m.breakdownProbability {
		m.setState(StateFault)
		m.faultStart = time.Now()
	}
}

func (m *BaseMachine) tickFault() {
	if m.autoRestoreTimeMS <= 0 {
		return
	}
	elapsed := time.Since(m.faultStart).Milliseconds()
	effectiveRestore := float64(m.autoRestoreTimeMS) / m.timeScale
	if float64(elapsed) >= effectiveRestore {
		m.setState(StateIdle)
		// Auto-restart if we have an active order
		if m.currentOrderID != "" {
			m.setState(StateRunning)
		}
	}
}

func (m *BaseMachine) effectiveCycleTimeMS() float64 {
	base := float64(m.cycleTimeMS)
	// Apply random variance
	variance := base * m.randomFactor
	base += (m.rng.Float64()*2 - 1) * variance
	// Apply time scale
	if m.timeScale > 0 {
		base /= m.timeScale
	}
	if base < 1 {
		base = 1
	}
	return base
}

func (m *BaseMachine) canTransition(target State) bool {
	allowed, ok := ValidTransitions[m.state]
	if !ok {
		return false
	}
	for _, s := range allowed {
		if s == target {
			return true
		}
	}
	return false
}

func (m *BaseMachine) setState(newState State) {
	now := time.Now()
	oldState := m.state

	// Track running time
	if m.state == StateRunning {
		if !m.runStart.IsZero() {
			m.runTimeNS += now.Sub(m.runStart).Nanoseconds()
		}
	} else if m.state == StateFault || m.state == StateStopped || m.state == StateMaintenance {
		if !m.downStart.IsZero() {
			m.downTimeNS += now.Sub(m.downStart).Nanoseconds()
		}
	}

	m.state = newState

	if newState == StateRunning {
		m.runStart = now
	} else if newState == StateFault || newState == StateStopped || newState == StateMaintenance {
		m.downStart = now
	}

	// Publish state change event
	if m.eventBus != nil && oldState != newState {
		bus := m.eventBus
		id := m.id
		lineID := m.lineInstanceID
		m.mu.Unlock()
		bus.Publish(event.Event{
			Type:      event.MachineStateChange,
			MachineID: id,
			LineID:    lineID,
			Data: map[string]interface{}{
				"previous_state": string(oldState),
				"new_state":      string(newState),
			},
		})
		m.mu.Lock()
	}
}

// Rng exposes the machine's RNG for subtypes.
func (m *BaseMachine) Rng() *rand.Rand { return m.rng }

// Locked access helpers for subtypes that need to read base fields while holding their own lock.
func (m *BaseMachine) GetCycleTimeMS() int          { return m.cycleTimeMS }
func (m *BaseMachine) GetScrapRate() float64         { return m.scrapRate }
func (m *BaseMachine) GetTimeScale() float64         { return m.timeScale }
func (m *BaseMachine) GetRandomFactor() float64      { return m.randomFactor }
func (m *BaseMachine) GetStateUnlocked() State       { return m.state }
func (m *BaseMachine) GetCycleCount() int64          { return m.cycleCount }
func (m *BaseMachine) GetGoodCount() int64           { return m.goodCount }
func (m *BaseMachine) GetScrapCount() int64          { return m.scrapCount }
func (m *BaseMachine) GetBlockedByBuffer() bool      { return m.blockedByBuffer }
func (m *BaseMachine) Lock()                         { m.mu.Lock() }
func (m *BaseMachine) Unlock()                       { m.mu.Unlock() }
func (m *BaseMachine) RLock()                        { m.mu.RLock() }
func (m *BaseMachine) RUnlock()                      { m.mu.RUnlock() }
