package mes

import (
	"context"
	"math/rand"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"machine-simulator-2/internal/config"
	"machine-simulator-2/internal/event"
	"machine-simulator-2/internal/line"
	"machine-simulator-2/internal/machine"
	"machine-simulator-2/internal/recipe"
	"machine-simulator-2/internal/storage"
)

// Config holds MES simulator settings.
type Config struct {
	DispatchIntervalMS int
	DispatchStrategy   string
	TimeScale          float64
}

// LineRef holds references needed to dispatch to a line.
type LineRef struct {
	InstanceID         string
	TemplateName       string
	MachineIDs         []string
	Stages             []int
	DefaultCycleTimeMS []int
}

// MES simulates manufacturing execution - dispatch and progress tracking.
type MES struct {
	mu sync.RWMutex

	cfg        Config
	orders     *storage.OrderRepository
	operations *storage.OperationRepository
	bus        *event.Bus
	catalog    *recipe.Catalog
	lineMgr    *line.Manager
	engine     engineAccessor

	lines         map[string]*LineRef // line instance ID -> ref
	machineToLine map[string]string   // machine ID -> line instance ID

	// Round-robin tracking
	dispatchCounters map[string]int // template -> last dispatched index
	instancesByTemplate map[string][]string // template -> []instanceID

	rng *rand.Rand
}

// engineAccessor is the subset of simulation.Engine we need.
type engineAccessor interface {
	GetMachine(id string) machine.Machine
}

// New creates a new MES simulator.
func New(cfg Config, orders *storage.OrderRepository, ops *storage.OperationRepository,
	bus *event.Bus, catalog *recipe.Catalog, lineMgr *line.Manager, engine engineAccessor) *MES {
	return &MES{
		cfg:                 cfg,
		orders:              orders,
		operations:          ops,
		bus:                 bus,
		catalog:             catalog,
		lineMgr:             lineMgr,
		engine:              engine,
		lines:               make(map[string]*LineRef),
		machineToLine:       make(map[string]string),
		dispatchCounters:    make(map[string]int),
		instancesByTemplate: make(map[string][]string),
		rng:                 rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// RegisterLine adds a line instance for MES dispatch.
func (m *MES) RegisterLine(li config.LineInstanceConfig) {
	m.mu.Lock()
	defer m.mu.Unlock()

	ref := &LineRef{
		InstanceID:   li.InstanceID,
		TemplateName: li.TemplateName,
	}
	for _, mc := range li.Machines {
		ref.MachineIDs = append(ref.MachineIDs, mc.ID)
		ref.Stages = append(ref.Stages, mc.Stage)
		ref.DefaultCycleTimeMS = append(ref.DefaultCycleTimeMS, mc.CycleTimeMS)
		m.machineToLine[mc.ID] = li.InstanceID
	}
	m.lines[li.InstanceID] = ref
	m.instancesByTemplate[li.TemplateName] = append(m.instancesByTemplate[li.TemplateName], li.InstanceID)
}

// SetTimeScale updates the time scale.
func (m *MES) SetTimeScale(scale float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cfg.TimeScale = scale
}

// Start begins the MES dispatch loop and subscribes to events.
func (m *MES) Start(ctx context.Context) {
	// Restore in-progress orders from a previous run (e.g. after Docker restart).
	// Machines start fresh with no order assignment, so we must re-assign persisted
	// IN_PROGRESS orders and kick-start the first machine of each line.
	m.restoreInProgressOrders()

	// Subscribe to cycle complete events for progress tracking
	m.bus.Subscribe(event.MachineCycleComplete, func(e event.Event) {
		m.handleCycleComplete(e.MachineID, e.Data.(bool))
	})

	// Subscribe to order created events for auto-release + dispatch
	m.bus.Subscribe(event.OrderCreated, func(e event.Event) {
		if order, ok := e.Data.(*storage.Order); ok {
			// Auto-release and dispatch
			go m.releaseAndDispatch(order)
		}
	})

	go m.dispatchLoop(ctx)
}

func (m *MES) dispatchLoop(ctx context.Context) {
	for {
		m.mu.RLock()
		intervalMS := m.cfg.DispatchIntervalMS
		scale := m.cfg.TimeScale
		m.mu.RUnlock()

		if scale <= 0 {
			scale = 1.0
		}
		interval := time.Duration(float64(intervalMS)/scale) * time.Millisecond

		select {
		case <-ctx.Done():
			return
		case <-time.After(interval):
			m.dispatchPendingOrders()
		}
	}
}

func (m *MES) dispatchPendingOrders() {
	orders, err := m.orders.List(storage.OrderFilter{Status: "RELEASED", Limit: 10})
	if err != nil {
		log.Error().Err(err).Msg("MES: failed to list released orders")
		return
	}

	for _, order := range orders {
		m.dispatchOrder(order)
	}
}

func (m *MES) releaseAndDispatch(order *storage.Order) {
	// Small delay to allow order to be committed
	time.Sleep(10 * time.Millisecond)

	if err := m.orders.UpdateStatus(order.ID, "RELEASED"); err != nil {
		log.Error().Err(err).Msg("MES: failed to release order")
		return
	}

	order.Status = "RELEASED"
	m.dispatchOrder(order)
}

func (m *MES) dispatchOrder(order *storage.Order) {
	m.mu.RLock()
	lineRef := m.lines[order.LineInstanceID]
	m.mu.RUnlock()

	if lineRef == nil {
		log.Warn().Str("line", order.LineInstanceID).Msg("MES: line not found for order")
		return
	}

	// Look up recipe for cycle time overrides
	r, recipeErr := m.catalog.GetRecipe(lineRef.TemplateName, order.ProductID)

	// Create operations for each stage
	for i, machineID := range lineRef.MachineIDs {
		stage := lineRef.Stages[i]
		opID := uuid.New().String()

		// Determine planned cycle time: recipe override takes priority, then machine default
		plannedCycleTimeMS := lineRef.DefaultCycleTimeMS[i]
		if recipeErr == nil {
			if ov, ok := r.StageOverrides[stage]; ok && ov.CycleTimeMS != nil {
				plannedCycleTimeMS = *ov.CycleTimeMS
			}
		}

		op := &storage.Operation{
			ID:                 opID,
			OrderID:            order.ID,
			MachineID:          machineID,
			Stage:              stage,
			Status:             "PENDING",
			PlannedCycleTimeMS: plannedCycleTimeMS,
		}

		if err := m.operations.Create(op); err != nil {
			log.Error().Err(err).Str("op", opID).Msg("MES: failed to create operation")
			continue
		}
	}

	// Update order status to IN_PROGRESS
	m.orders.UpdateStatus(order.ID, "IN_PROGRESS")

	// Apply recipe overrides
	if recipeErr == nil {
		for i, machineID := range lineRef.MachineIDs {
			stage := lineRef.Stages[i]
			if ov, ok := r.StageOverrides[stage]; ok {
				if mach := m.engine.GetMachine(machineID); mach != nil {
					mach.ApplyRecipeOverrides(ov.ToMachineOverrides())
				}
			}
		}
	}

	// Assign order to all machines, but only start the first
	ops, _ := m.operations.ListByOrderID(order.ID)
	for i, machineID := range lineRef.MachineIDs {
		mach := m.engine.GetMachine(machineID)
		if mach == nil || i >= len(ops) {
			continue
		}
		mach.AssignOrder(order.ID, ops[i].ID, order.ProductID)
		if i == 0 {
			mach.SendCommand(machine.CommandStart)
			m.operations.UpdateStatus(ops[i].ID, "IN_PROGRESS")
		}
	}

	log.Info().
		Str("order_id", order.ID).
		Str("line", order.LineInstanceID).
		Str("product", order.ProductID).
		Msg("MES: order dispatched")
}

func (m *MES) handleCycleComplete(machineID string, good bool) {
	m.mu.RLock()
	lineID := m.machineToLine[machineID]
	lineRef := m.lines[lineID]
	m.mu.RUnlock()

	if lineID == "" || lineRef == nil {
		return
	}

	// Route part to downstream buffer
	m.lineMgr.HandlePartComplete(machineID, good)

	// Try to start the downstream machine
	if good {
		m.tryStartDownstream(machineID, lineRef)
	}

	mach := m.engine.GetMachine(machineID)
	if mach == nil {
		return
	}

	// Non-first machines need upstream parts to continue cycling.
	// Try to pull the next part; if the buffer is empty, pause until fed.
	if !m.isFirstMachine(machineID, lineRef) {
		part := m.lineMgr.TryPullFromBuffer(machineID)
		if part == nil {
			mach.SetStarvedByBuffer(true)
		}
	}

	orderID, _ := mach.GetCurrentOrder()
	if orderID == "" {
		return
	}

	// Find the operation for this machine and order
	op, err := m.operations.GetByMachineAndOrder(machineID, orderID)
	if err != nil || op == nil {
		return
	}

	// Update operation quantities
	if good {
		op.GoodQty++
	} else {
		op.ScrapQty++
	}
	m.operations.UpdateQuantities(op.ID, op.GoodQty, op.ScrapQty)

	// Only update order-level quantities from the last machine (finished goods output)
	lastMachine := m.isLastMachine(machineID, lineRef)
	if !lastMachine {
		return
	}

	order, err := m.orders.GetByID(orderID)
	if err != nil {
		return
	}

	if good {
		m.orders.UpdateQuantities(orderID, order.GoodQty+1, order.ScrapQty)
	} else {
		m.orders.UpdateQuantities(orderID, order.GoodQty, order.ScrapQty+1)
	}

	// Publish progress event
	if m.bus != nil {
		m.bus.Publish(event.Event{
			Type:      event.OrderProgress,
			MachineID: machineID,
			LineID:    lineID,
			Data: map[string]interface{}{
				"order_id":             orderID,
				"good_qty":             order.GoodQty + boolToInt(good),
				"scrap_qty":            order.ScrapQty + boolToInt(!good),
				"planned_qty":          order.PlannedQty,
				"planned_cycle_time_ms": op.PlannedCycleTimeMS,
			},
		})
	}

	// Check if order complete based on last machine output
	currentGood := order.GoodQty + boolToInt(good)
	if currentGood >= order.PlannedQty {
		m.completeOrder(orderID)
	}
}

// tryStartDownstream starts the next machine in the line if it's idle.
func (m *MES) tryStartDownstream(machineID string, lineRef *LineRef) {
	idx := -1
	for i, id := range lineRef.MachineIDs {
		if id == machineID {
			idx = i
			break
		}
	}
	if idx < 0 || idx >= len(lineRef.MachineIDs)-1 {
		return
	}

	nextMachineID := lineRef.MachineIDs[idx+1]
	nextMachine := m.engine.GetMachine(nextMachineID)
	if nextMachine == nil {
		return
	}

	state := nextMachine.GetState()

	// Only pull from buffer when the downstream machine actually needs a part:
	// - IDLE: needs a part to start
	// - RUNNING + starved: waiting for upstream input to resume cycling
	// Do NOT pull when RUNNING and not starved — that machine is mid-cycle
	// and pulling would waste the part, keeping buffers perpetually empty.
	needsPart := state == machine.StateIdle || (state == machine.StateRunning && nextMachine.IsStarvedByBuffer())
	if !needsPart {
		return
	}

	part := m.lineMgr.TryPullFromBuffer(nextMachineID)
	if part == nil {
		return
	}

	nextMachine.SetStarvedByBuffer(false)

	if state == machine.StateIdle {
		nextMachine.SendCommand(machine.CommandStart)
	}

	// Mark the operation as in-progress
	orderID, _ := nextMachine.GetCurrentOrder()
	if orderID != "" {
		if op, err := m.operations.GetByMachineAndOrder(nextMachineID, orderID); err == nil && op != nil && op.Status == "PENDING" {
			m.operations.UpdateStatus(op.ID, "IN_PROGRESS")
		}
	}
}

// isFirstMachine returns true if machineID is the first machine in the line.
func (m *MES) isFirstMachine(machineID string, lineRef *LineRef) bool {
	return len(lineRef.MachineIDs) > 0 && lineRef.MachineIDs[0] == machineID
}

// isLastMachine returns true if machineID is the last machine in the line.
func (m *MES) isLastMachine(machineID string, lineRef *LineRef) bool {
	return len(lineRef.MachineIDs) > 0 && lineRef.MachineIDs[len(lineRef.MachineIDs)-1] == machineID
}

func (m *MES) completeOrder(orderID string) {
	// Mark all operations as completed
	ops, _ := m.operations.ListByOrderID(orderID)
	for _, op := range ops {
		if op.Status != "COMPLETED" {
			m.operations.UpdateStatus(op.ID, "COMPLETED")
		}
	}

	// Close the order
	m.orders.UpdateStatus(orderID, "CLOSED")

	order, _ := m.orders.GetByID(orderID)
	if order == nil {
		return
	}

	// Revert recipe overrides
	m.mu.RLock()
	lineRef := m.lines[order.LineInstanceID]
	m.mu.RUnlock()

	if lineRef != nil {
		for _, machineID := range lineRef.MachineIDs {
			if mach := m.engine.GetMachine(machineID); mach != nil {
				mach.ClearOrder()
			}
		}
	}

	if m.bus != nil {
		m.bus.Publish(event.Event{
			Type:   event.OrderClosed,
			LineID: order.LineInstanceID,
			Data:   order,
		})
	}

	log.Info().
		Str("order_id", orderID).
		Int("good_qty", order.GoodQty).
		Msg("MES: order completed")
}

// restoreInProgressOrders re-assigns persisted IN_PROGRESS orders to machines
// after a restart. On a fresh start with an empty database this is a no-op.
func (m *MES) restoreInProgressOrders() {
	orders, err := m.orders.List(storage.OrderFilter{Status: "IN_PROGRESS"})
	if err != nil || len(orders) == 0 {
		return
	}

	restored := 0
	for _, order := range orders {
		m.mu.RLock()
		lineRef := m.lines[order.LineInstanceID]
		m.mu.RUnlock()
		if lineRef == nil {
			continue
		}

		ops, err := m.operations.ListByOrderID(order.ID)
		if err != nil || len(ops) == 0 {
			continue
		}

		// Build machine->operation lookup from persisted operations
		opByMachine := make(map[string]*storage.Operation, len(ops))
		for _, op := range ops {
			opByMachine[op.MachineID] = op
		}

		// Re-apply recipe overrides
		r, recipeErr := m.catalog.GetRecipe(lineRef.TemplateName, order.ProductID)
		if recipeErr == nil {
			for i, machineID := range lineRef.MachineIDs {
				stage := lineRef.Stages[i]
				if ov, ok := r.StageOverrides[stage]; ok {
					if mach := m.engine.GetMachine(machineID); mach != nil {
						mach.ApplyRecipeOverrides(ov.ToMachineOverrides())
					}
				}
			}
		}

		// Re-assign order to all machines and start the first one
		for i, machineID := range lineRef.MachineIDs {
			mach := m.engine.GetMachine(machineID)
			if mach == nil {
				continue
			}
			op := opByMachine[machineID]
			if op == nil {
				continue
			}
			mach.AssignOrder(order.ID, op.ID, order.ProductID)
			if i == 0 {
				mach.SendCommand(machine.CommandStart)
			}
		}
		restored++
	}

	if restored > 0 {
		log.Info().Int("count", restored).Msg("MES: restored in-progress orders from previous run")
	}
}

// CancelOrder stops an active order and reverts overrides.
func (m *MES) CancelOrder(orderID string) error {
	order, err := m.orders.GetByID(orderID)
	if err != nil {
		return err
	}

	// Mark operations as cancelled
	ops, _ := m.operations.ListByOrderID(orderID)
	for _, op := range ops {
		if op.Status != "COMPLETED" {
			m.operations.UpdateStatus(op.ID, "CANCELLED")
		}
	}

	// Cancel the order
	m.orders.UpdateStatus(orderID, "CANCELLED")

	// Revert recipe overrides and clear assignments
	m.mu.RLock()
	lineRef := m.lines[order.LineInstanceID]
	m.mu.RUnlock()

	if lineRef != nil {
		for _, machineID := range lineRef.MachineIDs {
			if mach := m.engine.GetMachine(machineID); mach != nil {
				mach.ClearOrder()
			}
		}
	}

	return nil
}

// SelectLineInstance picks a line instance for dispatch based on strategy.
func (m *MES) SelectLineInstance(templateName string) string {
	m.mu.Lock()
	defer m.mu.Unlock()

	instances := m.instancesByTemplate[templateName]
	if len(instances) == 0 {
		return ""
	}
	if len(instances) == 1 {
		return instances[0]
	}

	switch m.cfg.DispatchStrategy {
	case "round-robin":
		idx := m.dispatchCounters[templateName] % len(instances)
		m.dispatchCounters[templateName] = idx + 1
		return instances[idx]
	case "least-busy":
		minPending := int(^uint(0) >> 1)
		selected := instances[0]
		for _, id := range instances {
			count, _ := m.orders.CountByStatus(id, "IN_PROGRESS")
			if count < minPending {
				minPending = count
				selected = id
			}
		}
		return selected
	case "priority":
		// Priority: prefer earlier lines, fill sequentially.
		// Select the first line with no in-progress orders; if all busy, pick least-busy.
		minPending := int(^uint(0) >> 1)
		selected := instances[0]
		for _, id := range instances {
			count, _ := m.orders.CountByStatus(id, "IN_PROGRESS")
			if count == 0 {
				return id
			}
			if count < minPending {
				minPending = count
				selected = id
			}
		}
		return selected
	default:
		return instances[m.rng.Intn(len(instances))]
	}
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
