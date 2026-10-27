package line

import (
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"machine-simulator-2/internal/event"
	"machine-simulator-2/internal/machine"
)

// JobState is the state of the job loaded on a line.
type JobState string

const (
	JobNone          JobState = "NO_JOB"
	JobLoaded        JobState = "LOADED"
	JobRunning       JobState = "RUNNING"
	JobInterrupted   JobState = "INTERRUPTED"
	JobTargetReached JobState = "TARGET_REACHED"
)

// Errors returned by the job commands. Callers branch on them to pick a
// response code (REST) or a command result (OPC UA).
var (
	ErrUnknownLine       = errors.New("unknown line")
	ErrInvalidTransition = errors.New("invalid job transition")
	ErrUnknownProduct    = errors.New("unknown product")
	ErrInvalidJob        = errors.New("invalid job")
)

// JobSpec is what an MES (or operator) loads onto a line.
type JobSpec struct {
	OrderID     string `json:"order_id"`
	OperationID string `json:"operation_id"`
	ProductID   string `json:"product_id"`
	// TargetQty is the PLC batch counter: stage 1 stops releasing parts once
	// good + in-process parts reach it. 0 means open-ended.
	TargetQty int `json:"target_qty"`
	// InitialGood seeds the good counter, used when a job is restored after a
	// restart.
	InitialGood int `json:"initial_good,omitempty"`
	// Components stage 1 takes from the line-side store for every part it
	// starts. None: the job needs no material.
	Components []Component `json:"components,omitempty"`
}

// Component is one material a part consumes at stage 1.
type Component struct {
	Material string  `json:"material"`
	PerPart  float64 `json:"per_part"`
}

// JobStatus is a snapshot of the line's job.
type JobStatus struct {
	JobSpec
	State          JobState `json:"state"`
	Good           int      `json:"good"`
	Scrap          int      `json:"scrap"`
	Started        int      `json:"started"`
	PartsInProcess int      `json:"parts_in_process"`
	// MaterialMissing is the component stage 1 waits for, if any.
	MaterialMissing string `json:"material_missing,omitempty"`
	// Material is the line-side store: quantity per material.
	Material map[string]float64 `json:"material,omitempty"`
}

// Ledger counts every part a line has started since the simulator started.
// Balanced() must always hold: parts are never created or lost silently.
type Ledger struct {
	Started         int         `json:"started"`
	DeliveredGood   int         `json:"delivered_good"`
	Scrapped        int         `json:"scrapped"`
	ScrappedByStage map[int]int `json:"scrapped_by_stage"`
	WIPScrapped     int         `json:"wip_scrapped"`
	WIPCarriedOver  int         `json:"wip_carried_over"`
	InBuffers       int         `json:"in_buffers"`
	InMachines      int         `json:"in_machines"`
}

// Balanced reports whether every started part is accounted for.
func (l Ledger) Balanced() bool {
	return l.Started == l.DeliveredGood+l.Scrapped+l.WIPScrapped+l.InBuffers+l.InMachines
}

// PartBooked is the data of a line.part_booked event: one finished cycle of a
// stage, attributed to the order of the part it processed.
type PartBooked struct {
	MachineID string `json:"machine_id"`
	OrderID   string `json:"order_id"`
	Stage     int    `json:"stage"`
	Good      bool   `json:"good"`
	LastStage bool   `json:"last_stage"`
}

// RecipeSource resolves a product's per-stage overrides (keyed by stage number).
type RecipeSource interface {
	StageOverrides(templateName, productID string) (map[int]machine.RecipeOverrides, error)
}

type jobRun struct {
	id      string
	spec    JobSpec
	state   JobState
	good    int
	scrap   int
	started int
}

// flow holds the material flow of one line: which part sits in which machine
// or buffer, the loaded job and the ledger. All part movements happen under mu.
//
// Lock order: flow.mu → machine lock → buffer lock. Machines release their own
// lock before calling the cycle-complete callback, so this order is never
// inverted.
type flow struct {
	mu sync.Mutex

	line     *Line
	machines []machine.Machine // by stage index
	buffers  []*Buffer         // buffers[i] sits between stage i and i+1

	inWork []*Part // part being processed per stage
	held   []*Part // finished good part waiting for space downstream

	job    *jobRun
	ledger Ledger

	// material is the line-side store. It belongs to the line, not the job,
	// so leftovers serve the next order.
	material        map[string]float64
	materialMissing string

	lastProgress time.Time
	pending      []event.Event // published after mu is released
}

func newFlow(l *Line, machines map[string]machine.Machine) *flow {
	n := len(l.MachineIDs)
	f := &flow{
		line:         l,
		machines:     make([]machine.Machine, n),
		buffers:      l.OrderedBuffers(),
		inWork:       make([]*Part, n),
		held:         make([]*Part, n),
		ledger:       Ledger{ScrappedByStage: make(map[int]int)},
		material:     make(map[string]float64),
		lastProgress: time.Now(),
	}
	for i, id := range l.MachineIDs {
		m := machines[id]
		f.machines[i] = m
		if m == nil {
			continue
		}
		// A machine only cycles while it holds a part.
		m.SetStarvedByBuffer(true)
		if g, ok := m.(interface {
			SetFlowBuffers(in, out machine.BufferGauge)
		}); ok {
			var in, out machine.BufferGauge
			if i > 0 && i-1 < len(f.buffers) {
				in = f.buffers[i-1]
			}
			if i < len(f.buffers) {
				out = f.buffers[i]
			}
			g.SetFlowBuffers(in, out)
		}
	}
	return f
}

func (f *flow) stageNumber(i int) int {
	if i < len(f.line.Stages) {
		return f.line.Stages[i]
	}
	return i + 1
}

func (f *flow) stageOf(machineID string) int {
	for i, id := range f.line.MachineIDs {
		if id == machineID {
			return i
		}
	}
	return -1
}

// downstream returns the buffer after stage i, or nil for the last stage.
func (f *flow) downstream(i int) *Buffer {
	if i < len(f.buffers) {
		return f.buffers[i]
	}
	return nil
}

func (f *flow) upstream(i int) *Buffer {
	if i > 0 && i-1 < len(f.buffers) {
		return f.buffers[i-1]
	}
	return nil
}

// eachPart calls fn for every part in the line.
func (f *flow) eachPart(fn func(p *Part)) {
	for i := range f.inWork {
		if f.inWork[i] != nil {
			fn(f.inWork[i])
		}
		if f.held[i] != nil {
			fn(f.held[i])
		}
	}
	for _, b := range f.buffers {
		b.forEach(fn)
	}
}

func (f *flow) inProcess(jobID string) int {
	n := 0
	f.eachPart(func(p *Part) {
		if p.JobID == jobID {
			n++
		}
	})
	return n
}

func (f *flow) snapshotLedger() Ledger {
	l := f.ledger
	l.ScrappedByStage = make(map[int]int, len(f.ledger.ScrappedByStage))
	for k, v := range f.ledger.ScrappedByStage {
		l.ScrappedByStage[k] = v
	}
	l.InBuffers, l.InMachines = 0, 0
	for i := range f.inWork {
		if f.inWork[i] != nil {
			l.InMachines++
		}
		if f.held[i] != nil {
			l.InMachines++
		}
	}
	for _, b := range f.buffers {
		l.InBuffers += b.Count()
	}
	return l
}

func (f *flow) status() JobStatus {
	st := JobStatus{State: JobNone, Material: f.materialSnapshot()}
	if f.job == nil {
		return st
	}
	st.JobSpec = f.job.spec
	st.State = f.job.state
	st.Good, st.Scrap, st.Started = f.job.good, f.job.scrap, f.job.started
	st.PartsInProcess = f.inProcess(f.job.id)
	st.MaterialMissing = f.materialMissing
	return st
}

func (f *flow) materialSnapshot() map[string]float64 {
	if len(f.material) == 0 {
		return nil
	}
	out := make(map[string]float64, len(f.material))
	for k, v := range f.material {
		out[k] = v
	}
	return out
}

// shortOf returns the first component the line-side store cannot supply for
// one more part, or "".
func (f *flow) shortOf() string {
	for _, c := range f.job.spec.Components {
		if f.material[c.Material]+1e-9 < c.PerPart {
			return c.Material
		}
	}
	return ""
}

func (f *flow) emitJobChanged() {
	f.pending = append(f.pending, event.Event{
		Type:   event.LineJobChanged,
		LineID: f.line.InstanceID,
		Data:   f.status(),
	})
}

// takePending returns and clears events queued under mu.
func (f *flow) takePending() []event.Event {
	ev := f.pending
	f.pending = nil
	return ev
}

// gateOpen reports whether stage 1 may start another part.
func (f *flow) gateOpen() bool {
	j := f.job
	if j == nil || j.state != JobRunning {
		return false
	}
	return j.spec.TargetQty <= 0 || j.good+f.inProcess(j.id) < j.spec.TargetQty
}

// acquire gives stage i its next part if one is available.
func (f *flow) acquire(i int) {
	if f.inWork[i] != nil || f.held[i] != nil {
		return
	}
	if i == 0 {
		if !f.gateOpen() {
			return
		}
		short := f.shortOf()
		if short != f.materialMissing {
			f.materialMissing = short
			f.emitJobChanged()
		}
		if short != "" {
			return
		}
		for _, c := range f.job.spec.Components {
			f.material[c.Material] -= c.PerPart
		}
		f.inWork[0] = &Part{
			ID:        uuid.New().String(),
			JobID:     f.job.id,
			OrderID:   f.job.spec.OrderID,
			ProductID: f.job.spec.ProductID,
			CreatedAt: time.Now(),
		}
		f.ledger.Started++
		f.job.started++
		return
	}
	if up := f.upstream(i); up != nil {
		f.inWork[i] = up.Remove()
	}
}

// kick moves parts as far as they can go and sets machine flags to match.
// startIdle sends START to idle machines that now hold a part.
func (f *flow) kick(startIdle bool) {
	// Last stage first: pulling frees buffer space for the stage before.
	for i := len(f.machines) - 1; i >= 0; i-- {
		if f.held[i] != nil {
			if down := f.downstream(i); down != nil && down.Add(f.held[i]) {
				f.held[i] = nil
				f.partMoved(i, down)
			}
		}
		f.acquire(i)
	}
	for i, m := range f.machines {
		if m == nil {
			continue
		}
		m.SetBlockedByBuffer(f.held[i] != nil)
		m.SetStarvedByBuffer(f.inWork[i] == nil && f.held[i] == nil)
		if !startIdle || f.inWork[i] == nil {
			continue
		}
		switch m.GetState() {
		case machine.StateIdle:
			_ = m.SendCommand(machine.CommandStart)
		case machine.StateStopped:
			_ = m.SendCommand(machine.CommandReset)
			_ = m.SendCommand(machine.CommandStart)
		}
	}
}

func (f *flow) partMoved(i int, down *Buffer) {
	f.pending = append(f.pending, event.Event{
		Type:   event.LinePartMoved,
		LineID: f.line.InstanceID,
		Data: map[string]interface{}{
			"from_machine": f.line.MachineIDs[i],
			"buffer_id":    down.ID,
			"buffer_count": down.Count(),
		},
	})
}

func (f *flow) jobActive() bool {
	return f.job != nil && f.job.state != JobLoaded
}

// cycleComplete books the part stage i just finished and moves material on.
func (f *flow) cycleComplete(i int, good bool) {
	p := f.inWork[i]
	if p == nil {
		// A machine cycled without a part (started outside a job). Nothing to book.
		log.Debug().Str("line", f.line.InstanceID).Int("stage", i+1).Msg("cycle without part ignored")
		return
	}
	f.inWork[i] = nil
	f.lastProgress = time.Now()
	ownJob := f.job != nil && p.JobID == f.job.id
	f.pending = append(f.pending, event.Event{
		Type:      event.LinePartBooked,
		MachineID: f.line.MachineIDs[i],
		LineID:    f.line.InstanceID,
		Data: PartBooked{
			MachineID: f.line.MachineIDs[i],
			OrderID:   p.OrderID,
			Stage:     f.stageNumber(i),
			Good:      good,
			LastStage: i == len(f.machines)-1,
		},
	})

	switch {
	case !good:
		f.ledger.Scrapped++
		f.ledger.ScrappedByStage[f.stageNumber(i)]++
		if ownJob {
			f.job.scrap++
		}
	case i == len(f.machines)-1:
		f.ledger.DeliveredGood++
		if ownJob {
			f.job.good++
		}
	default:
		p.IsGood = true
		if down := f.downstream(i); down.Add(p) {
			f.partMoved(i, down)
		} else {
			f.held[i] = p
		}
	}

	f.kick(f.jobActive())
	f.checkTarget()
}

func (f *flow) checkTarget() {
	j := f.job
	if j == nil || (j.state != JobRunning && j.state != JobInterrupted) || j.spec.TargetQty <= 0 {
		return
	}
	if j.good >= j.spec.TargetQty && f.inProcess(j.id) == 0 {
		j.state = JobTargetReached
		f.emitJobChanged()
	}
}

// scrapParts takes every part matching match out of the line and counts it
// as WIP scrap.
func (f *flow) scrapParts(match func(p *Part) bool) int {
	n := 0
	for i := range f.inWork {
		if f.inWork[i] != nil && match(f.inWork[i]) {
			f.inWork[i] = nil
			n++
		}
		if f.held[i] != nil && match(f.held[i]) {
			f.held[i] = nil
			n++
		}
	}
	for _, b := range f.buffers {
		n += b.removeIf(match)
	}
	f.ledger.WIPScrapped += n
	return n
}

// Manager methods for jobs and flow.

func (mgr *Manager) flowFor(lineID string) (*flow, error) {
	mgr.mu.RLock()
	defer mgr.mu.RUnlock()
	f, ok := mgr.flows[lineID]
	if !ok {
		return nil, fmt.Errorf("%w %q", ErrUnknownLine, lineID)
	}
	return f, nil
}

func (mgr *Manager) publish(events []event.Event) {
	if mgr.bus == nil {
		return
	}
	for _, e := range events {
		mgr.bus.Publish(e)
	}
}

// SetRecipeSource sets where product recipes come from when a job is loaded.
func (mgr *Manager) SetRecipeSource(rs RecipeSource) {
	mgr.mu.Lock()
	defer mgr.mu.Unlock()
	mgr.recipes = rs
}

// SubscribeCycleEvents routes machine cycle completions into the line flows.
// Call it before other subscribers to machine.cycle_complete so they see the
// part already booked.
func (mgr *Manager) SubscribeCycleEvents() {
	if mgr.bus == nil {
		return
	}
	mgr.bus.Subscribe(event.MachineCycleComplete, func(e event.Event) {
		good, ok := e.Data.(bool)
		if !ok {
			return
		}
		mgr.OnCycleComplete(e.MachineID, good)
	})
}

// OnCycleComplete books a finished cycle of a machine.
func (mgr *Manager) OnCycleComplete(machineID string, good bool) {
	lineID := mgr.GetLineForMachine(machineID)
	f, err := mgr.flowFor(lineID)
	if err != nil {
		return
	}
	f.mu.Lock()
	if i := f.stageOf(machineID); i >= 0 {
		f.cycleComplete(i, good)
	}
	ev := f.takePending()
	f.mu.Unlock()
	mgr.publish(ev)
}

// LoadJob loads a job onto a line. Parts left in the line from an earlier job
// are carried over when they are the same product and scrapped otherwise.
func (mgr *Manager) LoadJob(lineID string, spec JobSpec) (JobStatus, error) {
	f, err := mgr.flowFor(lineID)
	if err != nil {
		return JobStatus{}, err
	}
	if spec.OrderID == "" || spec.ProductID == "" || spec.TargetQty < 0 {
		return JobStatus{}, fmt.Errorf("%w: order_id and product_id are required, target_qty must be >= 0", ErrInvalidJob)
	}
	for _, c := range spec.Components {
		if c.Material == "" || c.PerPart <= 0 {
			return JobStatus{}, fmt.Errorf("%w: component %q needs a material and a positive per_part", ErrInvalidJob, c.Material)
		}
	}
	if err := f.line.ValidateProduct(spec.ProductID); err != nil {
		return JobStatus{}, err
	}

	mgr.mu.RLock()
	rs := mgr.recipes
	mgr.mu.RUnlock()
	var overrides map[int]machine.RecipeOverrides
	if rs != nil {
		if overrides, err = rs.StageOverrides(f.line.TemplateName, spec.ProductID); err != nil {
			return JobStatus{}, fmt.Errorf("%w %q: %v", ErrUnknownProduct, spec.ProductID, err)
		}
	}

	f.mu.Lock()
	if f.job != nil && f.job.state != JobLoaded {
		st := f.job.state
		f.mu.Unlock()
		return JobStatus{}, fmt.Errorf("%w: load while %s, finish the job first", ErrInvalidTransition, st)
	}
	replacedID := ""
	if f.job != nil {
		replacedID = f.job.id // a loaded job that never started is replaced
	}
	j := &jobRun{id: uuid.New().String(), spec: spec, state: JobLoaded, good: spec.InitialGood}
	f.job = j
	f.materialMissing = ""

	// Carry over leftover parts of the same product, but no more than the job
	// still needs; the rest is purged below like another product's parts.
	need := -1
	if spec.TargetQty > 0 {
		need = max(spec.TargetQty-spec.InitialGood, 0)
	}
	carried, assigned := 0, 0
	f.eachPart(func(p *Part) {
		if p.ProductID != spec.ProductID || assigned == need {
			return
		}
		if p.JobID != replacedID {
			carried++
		}
		p.JobID, p.OrderID = j.id, spec.OrderID
		assigned++
	})
	f.ledger.WIPCarriedOver += carried
	if purged := f.scrapParts(func(p *Part) bool { return p.JobID != j.id }); purged > 0 {
		log.Info().Str("line", lineID).Int("parts", purged).Msg("changeover: leftover parts scrapped")
	}

	for i, m := range f.machines {
		if m == nil {
			continue
		}
		m.ClearOrder() // reverts the previous recipe
		if ov, ok := overrides[f.stageNumber(i)]; ok {
			m.ApplyRecipeOverrides(ov)
		}
		m.AssignOrder(spec.OrderID, spec.OperationID, spec.ProductID)
	}
	f.kick(false)
	f.emitJobChanged()
	st := f.status()
	ev := f.takePending()
	f.mu.Unlock()
	mgr.publish(ev)
	return st, nil
}

// StartJob starts or resumes the loaded job. Starting a running job is a no-op.
func (mgr *Manager) StartJob(lineID string) (JobStatus, error) {
	return mgr.command(lineID, func(f *flow) error {
		if f.job == nil {
			return fmt.Errorf("%w: no job loaded", ErrInvalidTransition)
		}
		switch f.job.state {
		case JobRunning:
			return nil
		case JobLoaded, JobInterrupted:
			f.job.state = JobRunning
			f.lastProgress = time.Now()
			f.kick(true)
			f.checkTarget()
			return nil
		default:
			return fmt.Errorf("%w: start while %s", ErrInvalidTransition, f.job.state)
		}
	})
}

// InterruptJob stops stage 1 from starting parts; parts in process finish.
func (mgr *Manager) InterruptJob(lineID string) (JobStatus, error) {
	return mgr.command(lineID, func(f *flow) error {
		if f.job == nil {
			return fmt.Errorf("%w: no job loaded", ErrInvalidTransition)
		}
		switch f.job.state {
		case JobInterrupted:
			return nil
		case JobRunning:
			f.job.state = JobInterrupted
			return nil
		default:
			return fmt.Errorf("%w: interrupt while %s", ErrInvalidTransition, f.job.state)
		}
	})
}

// FinishJob ends the job. Leftover parts stay in the line until the next load
// decides whether they are carried over or scrapped. The returned status is
// the job as it was finished.
func (mgr *Manager) FinishJob(lineID string) (JobStatus, error) {
	f, err := mgr.flowFor(lineID)
	if err != nil {
		return JobStatus{}, err
	}
	f.mu.Lock()
	if f.job == nil {
		f.mu.Unlock()
		return JobStatus{State: JobNone}, fmt.Errorf("%w: no job loaded", ErrInvalidTransition)
	}
	final := f.status()
	f.job = nil
	for _, m := range f.machines {
		if m != nil {
			m.ClearOrder()
		}
	}
	f.kick(false)
	f.emitJobChanged()
	ev := f.takePending()
	f.mu.Unlock()
	mgr.publish(ev)
	return final, nil
}

// ScrapWIP removes every part that does not belong to the current job and
// counts it as WIP scrap. Used when an order is cancelled.
func (mgr *Manager) ScrapWIP(lineID string) (int, error) {
	f, err := mgr.flowFor(lineID)
	if err != nil {
		return 0, err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	jobID := ""
	if f.job != nil {
		jobID = f.job.id
	}
	n := f.scrapParts(func(p *Part) bool { return p.JobID != jobID })
	f.kick(f.jobActive())
	return n, nil
}

func (mgr *Manager) command(lineID string, fn func(f *flow) error) (JobStatus, error) {
	f, err := mgr.flowFor(lineID)
	if err != nil {
		return JobStatus{}, err
	}
	f.mu.Lock()
	err = fn(f)
	if err == nil {
		f.emitJobChanged()
	}
	st := f.status()
	ev := f.takePending()
	f.mu.Unlock()
	mgr.publish(ev)
	return st, err
}

// Job returns the line's job.
func (mgr *Manager) Job(lineID string) (JobStatus, error) {
	f, err := mgr.flowFor(lineID)
	if err != nil {
		return JobStatus{}, err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.status(), nil
}

// Ledger returns the line's part ledger.
func (mgr *Manager) Ledger(lineID string) (Ledger, error) {
	f, err := mgr.flowFor(lineID)
	if err != nil {
		return Ledger{}, err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.snapshotLedger(), nil
}

// CheckStalls restarts lines whose job has had no part movement for longer
// than threshold although no machine is faulted. Buffers are never cleared.
// It returns the IDs of recovered lines.
func (mgr *Manager) CheckStalls(now time.Time, threshold time.Duration) []string {
	mgr.mu.RLock()
	ids := make([]string, 0, len(mgr.flows))
	for id := range mgr.flows {
		ids = append(ids, id)
	}
	mgr.mu.RUnlock()
	sort.Strings(ids)

	var recovered []string
	for _, id := range ids {
		f, err := mgr.flowFor(id)
		if err != nil {
			continue
		}
		f.mu.Lock()
		if f.stalled(now, threshold) {
			f.kick(true)
			f.lastProgress = now
			f.pending = append(f.pending, event.Event{Type: event.LineDeadlockRecovered, LineID: id})
			recovered = append(recovered, id)
		}
		ev := f.takePending()
		f.mu.Unlock()
		mgr.publish(ev)
	}
	return recovered
}

func (f *flow) stalled(now time.Time, threshold time.Duration) bool {
	if f.job == nil || (f.job.state != JobRunning && f.job.state != JobInterrupted) {
		return false
	}
	if now.Sub(f.lastProgress) < threshold {
		return false
	}
	hasWork := f.gateOpen()
	for i, m := range f.machines {
		if m == nil {
			continue
		}
		switch m.GetState() {
		case machine.StateFault, machine.StateMaintenance:
			// A fault is a normal stop that restores itself; not a stall.
			return false
		}
		if f.inWork[i] != nil || f.held[i] != nil {
			hasWork = true
		}
	}
	return hasWork
}

// AddMaterial puts material into the line-side store (staging from the
// warehouse) and lets a waiting stage 1 go on.
func (mgr *Manager) AddMaterial(lineID, material string, qty float64) (JobStatus, error) {
	if material == "" || qty <= 0 {
		return JobStatus{}, fmt.Errorf("%w: material and a positive quantity are required", ErrInvalidJob)
	}
	f, err := mgr.flowFor(lineID)
	if err != nil {
		return JobStatus{}, err
	}
	f.mu.Lock()
	f.material[material] += qty
	if f.job != nil && f.job.state == JobRunning {
		f.lastProgress = time.Now()
		f.kick(true)
	}
	if f.materialMissing != "" && f.job != nil && f.shortOf() == "" {
		f.materialMissing = ""
	}
	f.emitJobChanged()
	st := f.status()
	ev := f.takePending()
	f.mu.Unlock()
	mgr.publish(ev)
	return st, nil
}

// Material returns the line-side store.
func (mgr *Manager) Material(lineID string) (map[string]float64, error) {
	f, err := mgr.flowFor(lineID)
	if err != nil {
		return nil, err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.materialSnapshot(), nil
}

// RestoreMaterial replaces the line-side store, for loading it at start.
func (mgr *Manager) RestoreMaterial(lineID string, stock map[string]float64) error {
	f, err := mgr.flowFor(lineID)
	if err != nil {
		return err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.material = make(map[string]float64, len(stock))
	for k, v := range stock {
		f.material[k] = v
	}
	return nil
}
