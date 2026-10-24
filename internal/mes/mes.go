package mes

import (
	"context"
	"errors"
	"math/rand"
	"sort"
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

// lineJobs is the part of line.Manager the MES drives: it loads a job per
// order and reacts when the line reports the target reached.
type lineJobs interface {
	LoadJob(lineID string, spec line.JobSpec) (line.JobStatus, error)
	StartJob(lineID string) (line.JobStatus, error)
	FinishJob(lineID string) (line.JobStatus, error)
	ScrapWIP(lineID string) (int, error)
	Job(lineID string) (line.JobStatus, error)
}

// MES simulates manufacturing execution: it queues released orders per line,
// runs one order per line at a time as a line job, and tracks progress.
type MES struct {
	mu sync.RWMutex

	// dispatchMu serialises dispatch so an order is dispatched at most once.
	dispatchMu sync.Mutex

	cfg        Config
	orders     *storage.OrderRepository
	operations *storage.OperationRepository
	bus        *event.Bus
	catalog    *recipe.Catalog
	lineMgr    lineJobs
	engine     engineAccessor

	lines         map[string]*LineRef // line instance ID -> ref
	machineToLine map[string]string   // machine ID -> line instance ID

	// Round-robin tracking
	dispatchCounters    map[string]int      // template -> last dispatched index
	instancesByTemplate map[string][]string // template -> []instanceID

	rng *rand.Rand
}

// engineAccessor is the subset of simulation.Engine we need.
type engineAccessor interface {
	GetMachine(id string) machine.Machine
}

// New creates a new MES simulator.
func New(cfg Config, orders *storage.OrderRepository, ops *storage.OperationRepository,
	bus *event.Bus, catalog *recipe.Catalog, lineMgr lineJobs, engine engineAccessor) *MES {
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

// Start restores running orders, subscribes to events and starts the
// dispatch loop. Subscribe the line manager to cycle events first, so the
// MES counts parts the line has already booked.
func (m *MES) Start(ctx context.Context) {
	m.restoreInProgressOrders()

	m.bus.Subscribe(event.LinePartBooked, func(e event.Event) {
		if pb, ok := e.Data.(line.PartBooked); ok {
			m.handlePartBooked(e.LineID, pb)
		}
	})

	m.bus.Subscribe(event.LineJobChanged, func(e event.Event) {
		js, ok := e.Data.(line.JobStatus)
		if ok && js.State == line.JobTargetReached {
			m.completeOrder(js.OrderID)
		}
	})

	m.bus.Subscribe(event.OrderCreated, func(e event.Event) {
		if order, ok := e.Data.(*storage.Order); ok {
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

// dispatchPendingOrders is the safety net: it fills every idle line from its
// queue.
func (m *MES) dispatchPendingOrders() {
	m.mu.RLock()
	lineIDs := make([]string, 0, len(m.lines))
	for id := range m.lines {
		lineIDs = append(lineIDs, id)
	}
	m.mu.RUnlock()
	sort.Strings(lineIDs)

	for _, id := range lineIDs {
		m.tryDispatch(id)
	}
}

func (m *MES) releaseAndDispatch(order *storage.Order) {
	if err := m.orders.UpdateStatus(order.ID, "RELEASED"); err != nil {
		log.Error().Err(err).Msg("MES: failed to release order")
		return
	}
	m.tryDispatch(order.LineInstanceID)
}

// tryDispatch starts the next queued order of a line when the line has no
// job and no order in progress. Queue order: priority DESC, created_at ASC.
func (m *MES) tryDispatch(lineID string) {
	m.dispatchMu.Lock()
	defer m.dispatchMu.Unlock()

	js, err := m.lineMgr.Job(lineID)
	if err != nil || js.State != line.JobNone {
		return
	}
	if n, err := m.orders.CountByStatus(lineID, "IN_PROGRESS"); err != nil || n > 0 {
		return
	}
	queue, err := m.orders.List(storage.OrderFilter{Status: "RELEASED", LineInstanceID: lineID})
	if err != nil {
		log.Error().Err(err).Str("line", lineID).Msg("MES: failed to list released orders")
		return
	}
	if len(queue) == 0 {
		return
	}
	sort.SliceStable(queue, func(i, j int) bool {
		if queue[i].Priority != queue[j].Priority {
			return queue[i].Priority > queue[j].Priority
		}
		return queue[i].CreatedAt.Before(queue[j].CreatedAt)
	})
	m.dispatchOrder(queue[0])
}

// dispatchOrder creates the operations of an order and runs it as the line's
// job. Callers hold dispatchMu.
func (m *MES) dispatchOrder(order *storage.Order) {
	m.mu.RLock()
	lineRef := m.lines[order.LineInstanceID]
	m.mu.RUnlock()

	if lineRef == nil {
		log.Warn().Str("line", order.LineInstanceID).Msg("MES: line not found for order")
		return
	}

	r, recipeErr := m.catalog.GetRecipe(lineRef.TemplateName, order.ProductID)

	firstOpID := ""
	for i, machineID := range lineRef.MachineIDs {
		stage := lineRef.Stages[i]
		plannedCycleTimeMS := lineRef.DefaultCycleTimeMS[i]
		if recipeErr == nil {
			if ov, ok := r.StageOverrides[stage]; ok && ov.CycleTimeMS != nil {
				plannedCycleTimeMS = *ov.CycleTimeMS
			}
		}
		op := &storage.Operation{
			ID:                 uuid.New().String(),
			OrderID:            order.ID,
			MachineID:          machineID,
			Stage:              stage,
			Status:             "IN_PROGRESS",
			PlannedCycleTimeMS: plannedCycleTimeMS,
		}
		if err := m.operations.Create(op); err != nil {
			log.Error().Err(err).Str("op", op.ID).Msg("MES: failed to create operation")
			continue
		}
		if i == 0 {
			firstOpID = op.ID
		}
	}

	if err := m.runJob(order, firstOpID); err != nil {
		log.Error().Err(err).Str("order_id", order.ID).Msg("MES: line refused the order")
		return
	}
	if err := m.orders.UpdateStatus(order.ID, "IN_PROGRESS"); err != nil {
		log.Error().Err(err).Str("order_id", order.ID).Msg("MES: failed to set order IN_PROGRESS")
	}

	log.Info().
		Str("order_id", order.ID).
		Str("line", order.LineInstanceID).
		Str("product", order.ProductID).
		Msg("MES: order dispatched")
}

func (m *MES) runJob(order *storage.Order, opID string) error {
	_, err := m.lineMgr.LoadJob(order.LineInstanceID, line.JobSpec{
		OrderID:     order.ID,
		OperationID: opID,
		ProductID:   order.ProductID,
		TargetQty:   order.PlannedQty,
		InitialGood: order.GoodQty,
	})
	if err != nil {
		return err
	}
	_, err = m.lineMgr.StartJob(order.LineInstanceID)
	return err
}

// handlePartBooked updates operation and order quantities for a part the
// line has booked. Part routing is done by the line manager.
func (m *MES) handlePartBooked(lineID string, pb line.PartBooked) {
	if pb.OrderID == "" {
		return
	}
	op, err := m.operations.GetByMachineAndOrder(pb.MachineID, pb.OrderID)
	if err != nil || op == nil {
		return
	}
	if pb.Good {
		op.GoodQty++
	} else {
		op.ScrapQty++
	}
	m.operations.UpdateQuantities(op.ID, op.GoodQty, op.ScrapQty)

	if !pb.LastStage && pb.Good {
		return
	}

	// Order quantities come from the line job, which books every part under
	// one lock. Incrementing here would race between machines of the line.
	js, err := m.lineMgr.Job(lineID)
	if err != nil || js.OrderID != pb.OrderID {
		return
	}
	if err := m.orders.UpdateQuantities(pb.OrderID, js.Good, js.Scrap); err != nil {
		log.Error().Err(err).Str("order_id", pb.OrderID).Msg("MES: updating order quantities")
		return
	}
	order, err := m.orders.GetByID(pb.OrderID)
	if err != nil {
		return
	}
	if m.bus != nil {
		m.bus.Publish(event.Event{
			Type:      event.OrderProgress,
			MachineID: pb.MachineID,
			LineID:    lineID,
			Data: map[string]interface{}{
				"order_id":              pb.OrderID,
				"line_instance_id":      lineID,
				"good_qty":              js.Good,
				"scrap_qty":             js.Scrap,
				"started_qty":           js.Started,
				"planned_qty":           order.PlannedQty,
				"planned_cycle_time_ms": op.PlannedCycleTimeMS,
			},
		})
	}
}

func isLastMachine(machineID string, lineRef *LineRef) bool {
	return len(lineRef.MachineIDs) > 0 && lineRef.MachineIDs[len(lineRef.MachineIDs)-1] == machineID
}

// completeOrder closes an order whose line job reached its target and starts
// the next queued order.
func (m *MES) completeOrder(orderID string) {
	order, err := m.orders.GetByID(orderID)
	if err != nil || order == nil || order.Status != "IN_PROGRESS" {
		return
	}

	ops, _ := m.operations.ListByOrderID(orderID)
	for _, op := range ops {
		if op.Status != "COMPLETED" {
			m.operations.UpdateStatus(op.ID, "COMPLETED")
		}
	}
	final, err := m.lineMgr.FinishJob(order.LineInstanceID)
	if err != nil {
		log.Warn().Err(err).Str("order_id", orderID).Msg("MES: finishing line job")
	} else if final.OrderID == orderID {
		m.orders.UpdateQuantities(orderID, final.Good, final.Scrap)
	}
	m.orders.UpdateStatus(orderID, "CLOSED")

	if closed, err := m.orders.GetByID(orderID); err == nil {
		order = closed
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

	m.tryDispatch(order.LineInstanceID)
}

// restoreInProgressOrders resumes, per line, the most recently started
// IN_PROGRESS order and puts any others back into the queue.
func (m *MES) restoreInProgressOrders() {
	orders, err := m.orders.List(storage.OrderFilter{Status: "IN_PROGRESS"})
	if err != nil || len(orders) == 0 {
		return
	}

	byLine := make(map[string][]*storage.Order)
	for _, o := range orders {
		byLine[o.LineInstanceID] = append(byLine[o.LineInstanceID], o)
	}

	m.dispatchMu.Lock()
	defer m.dispatchMu.Unlock()
	for lineID, list := range byLine {
		sort.SliceStable(list, func(i, j int) bool { return startedAt(list[i]).After(startedAt(list[j])) })
		for _, o := range list[1:] {
			if err := m.orders.UpdateStatus(o.ID, "RELEASED"); err == nil {
				log.Warn().Str("order_id", o.ID).Str("line", lineID).Msg("MES: re-queued extra IN_PROGRESS order")
			}
		}

		keep := list[0]
		opID := ""
		if ops, err := m.operations.ListByOrderID(keep.ID); err == nil && len(ops) > 0 {
			opID = ops[0].ID
		}
		if err := m.runJob(keep, opID); err != nil {
			log.Error().Err(err).Str("order_id", keep.ID).Msg("MES: could not restore order")
			continue
		}
		log.Info().Str("order_id", keep.ID).Str("line", lineID).Msg("MES: restored in-progress order")
	}
}

func startedAt(o *storage.Order) time.Time {
	if o.StartedAt != nil {
		return *o.StartedAt
	}
	return o.CreatedAt
}

// CancelOrder cancels an order. If it is running, its line job ends and its
// work in progress is scrapped.
func (m *MES) CancelOrder(orderID string) error {
	order, err := m.orders.GetByID(orderID)
	if err != nil {
		return err
	}

	ops, _ := m.operations.ListByOrderID(orderID)
	for _, op := range ops {
		if op.Status != "COMPLETED" {
			m.operations.UpdateStatus(op.ID, "CANCELLED")
		}
	}
	if err := m.orders.UpdateStatus(orderID, "CANCELLED"); err != nil {
		return err
	}

	if js, err := m.lineMgr.Job(order.LineInstanceID); err == nil && js.OrderID == orderID {
		if _, err := m.lineMgr.FinishJob(order.LineInstanceID); err != nil && !errors.Is(err, line.ErrInvalidTransition) {
			return err
		}
		if _, err := m.lineMgr.ScrapWIP(order.LineInstanceID); err != nil {
			return err
		}
		go m.tryDispatch(order.LineInstanceID)
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
