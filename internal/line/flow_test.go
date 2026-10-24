package line

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"machine-simulator-2/internal/config"
	"machine-simulator-2/internal/event"
	"machine-simulator-2/internal/machine"
	"machine-simulator-2/internal/simulation"
)

// newTestLine builds a line of n stages with buffers of the given capacity.
// Machines are not started: tests drive cycles with OnCycleComplete.
func newTestLine(t *testing.T, n, capacity int) (*Manager, *Line, []*machine.BaseMachine) {
	t.Helper()
	li := config.LineInstanceConfig{
		InstanceID:   "l1",
		TemplateName: "tpl",
		Recipes: []config.RecipeConfig{
			{ProductID: "A"}, {ProductID: "B"},
		},
	}
	machines := make(map[string]machine.Machine)
	var bms []*machine.BaseMachine
	for i := 1; i <= n; i++ {
		mc := config.MachineInstanceConfig{
			ID: fmt.Sprintf("l1-s%d", i), Stage: i, CycleTimeMS: 100, LineInstanceID: "l1",
		}
		li.Machines = append(li.Machines, mc)
		bm := machine.NewBaseMachine(mc)
		machines[mc.ID] = bm
		bms = append(bms, bm)
		if i > 1 {
			li.Buffers = append(li.Buffers, config.BufferInstanceConfig{
				ID:                  fmt.Sprintf("b%d", i-1),
				UpstreamMachineID:   fmt.Sprintf("l1-s%d", i-1),
				DownstreamMachineID: mc.ID,
				Capacity:            capacity,
			})
		}
	}
	l := InstantiateFromConfig(li)
	mgr := NewManager(event.New())
	mgr.AddLine(l, machines)
	return mgr, l, bms
}

func mustLedger(t *testing.T, mgr *Manager) Ledger {
	t.Helper()
	led, err := mgr.Ledger("l1")
	if err != nil {
		t.Fatal(err)
	}
	if !led.Balanced() {
		t.Fatalf("ledger not balanced: %+v", led)
	}
	return led
}

func mustJob(t *testing.T, mgr *Manager) JobStatus {
	t.Helper()
	js, err := mgr.Job("l1")
	if err != nil {
		t.Fatal(err)
	}
	return js
}

// busyStages returns the stage indexes that hold a part in work.
func busyStages(mgr *Manager) []int {
	f, _ := mgr.flowFor("l1")
	f.mu.Lock()
	defer f.mu.Unlock()
	var s []int
	for i, p := range f.inWork {
		if p != nil {
			s = append(s, i)
		}
	}
	return s
}

// drive completes cycles, last busy stage first, until no stage holds a part
// or maxSteps is reached. scrapAt(step, stage) decides scrap.
func drive(t *testing.T, mgr *Manager, l *Line, maxSteps int, scrapAt func(step, stage int) bool) {
	t.Helper()
	for step := 0; step < maxSteps; step++ {
		busy := busyStages(mgr)
		if len(busy) == 0 {
			return
		}
		i := busy[len(busy)-1]
		mgr.OnCycleComplete(l.MachineIDs[i], !scrapAt(step, i))
		js := mustJob(t, mgr)
		mustLedger(t, mgr)
		if js.TargetQty > 0 && js.Good+js.PartsInProcess > js.TargetQty {
			t.Fatalf("step %d: over-release: good %d + in process %d > target %d", step, js.Good, js.PartsInProcess, js.TargetQty)
		}
	}
	t.Fatalf("line still busy after %d steps", maxSteps)
}

func noScrap(int, int) bool { return false }

func TestJobCommandsFollowStateMachine(t *testing.T) {
	type cmd func(*Manager) (JobStatus, error)
	load := func(m *Manager) (JobStatus, error) {
		return m.LoadJob("l1", JobSpec{OrderID: "o1", ProductID: "A", TargetQty: 5})
	}
	start := func(m *Manager) (JobStatus, error) { return m.StartJob("l1") }
	interrupt := func(m *Manager) (JobStatus, error) { return m.InterruptJob("l1") }
	finish := func(m *Manager) (JobStatus, error) { return m.FinishJob("l1") }

	tests := []struct {
		name      string
		setup     []cmd
		do        cmd
		wantState JobState
		wantErr   error
	}{
		{name: "load from no job", do: load, wantState: JobLoaded},
		{name: "start without job", do: start, wantState: JobNone, wantErr: ErrInvalidTransition},
		{name: "interrupt without job", do: interrupt, wantState: JobNone, wantErr: ErrInvalidTransition},
		{name: "finish without job", do: finish, wantState: JobNone, wantErr: ErrInvalidTransition},
		{name: "start loaded", setup: []cmd{load}, do: start, wantState: JobRunning},
		{name: "reload loaded", setup: []cmd{load}, do: load, wantState: JobLoaded},
		{name: "interrupt loaded", setup: []cmd{load}, do: interrupt, wantState: JobLoaded, wantErr: ErrInvalidTransition},
		{name: "load while running", setup: []cmd{load, start}, do: load, wantState: JobRunning, wantErr: ErrInvalidTransition},
		{name: "start running is no-op", setup: []cmd{load, start}, do: start, wantState: JobRunning},
		{name: "interrupt running", setup: []cmd{load, start}, do: interrupt, wantState: JobInterrupted},
		{name: "resume interrupted", setup: []cmd{load, start, interrupt}, do: start, wantState: JobRunning},
		{name: "finish running", setup: []cmd{load, start}, do: finish, wantState: JobNone},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr, _, _ := newTestLine(t, 2, 3)
			for _, c := range tt.setup {
				if _, err := c(mgr); err != nil {
					t.Fatalf("setup: %v", err)
				}
			}
			_, err := tt.do(mgr)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
			if got := mustJob(t, mgr).State; got != tt.wantState {
				t.Fatalf("state = %s, want %s", got, tt.wantState)
			}
		})
	}
}

func TestLoadRejectsUnknownProduct(t *testing.T) {
	mgr, _, _ := newTestLine(t, 2, 3)
	_, err := mgr.LoadJob("l1", JobSpec{OrderID: "o1", ProductID: "Z", TargetQty: 5})
	if !errors.Is(err, ErrUnknownProduct) {
		t.Fatalf("err = %v, want ErrUnknownProduct", err)
	}
	if got := mustJob(t, mgr).State; got != JobNone {
		t.Fatalf("state = %s, want NO_JOB", got)
	}
}

func TestTargetLimitsReleaseAndBalances(t *testing.T) {
	mgr, l, _ := newTestLine(t, 4, 2)
	if _, err := mgr.LoadJob("l1", JobSpec{OrderID: "o1", ProductID: "A", TargetQty: 20}); err != nil {
		t.Fatal(err)
	}
	if _, err := mgr.StartJob("l1"); err != nil {
		t.Fatal(err)
	}
	// Scrap every 7th completion at stage 3 and every 11th at stage 1.
	drive(t, mgr, l, 10000, func(step, stage int) bool {
		return (stage == 2 && step%7 == 0) || (stage == 0 && step%11 == 0)
	})

	js := mustJob(t, mgr)
	if js.State != JobTargetReached || js.Good != 20 {
		t.Fatalf("job = %+v, want TARGET_REACHED with 20 good", js)
	}
	led := mustLedger(t, mgr)
	if led.Started != js.Good+js.Scrap {
		t.Fatalf("started %d, want good %d + scrap %d", led.Started, js.Good, js.Scrap)
	}
	if led.InBuffers+led.InMachines != 0 {
		t.Fatalf("line not empty at target: %+v", led)
	}
}

func TestFullBufferHoldsPart(t *testing.T) {
	mgr, l, bms := newTestLine(t, 2, 1)
	if _, err := mgr.LoadJob("l1", JobSpec{OrderID: "o1", ProductID: "A"}); err != nil {
		t.Fatal(err)
	}
	if _, err := mgr.StartJob("l1"); err != nil {
		t.Fatal(err)
	}
	// Stage 2 never finishes: stage 1 fills the buffer, then the in-work part of
	// stage 2, then holds one more.
	// 1st part goes through the buffer into stage 2, 2nd waits in the buffer,
	// 3rd is held at stage 1, which then gets no new part.
	for k := 0; k < 3; k++ {
		mgr.OnCycleComplete(l.MachineIDs[0], true)
		mustLedger(t, mgr)
	}
	led := mustLedger(t, mgr)
	if led.Started != 3 || led.InBuffers != 1 || led.InMachines != 2 {
		t.Fatalf("ledger = %+v, want 3 started: 1 buffered, 2 in machines", led)
	}
	if !bms[0].IsBlockedByBuffer() {
		t.Fatal("stage 1 should be blocked while holding a finished part")
	}
	// Stage 2 finishes: the buffered part moves in, the held part moves to the buffer.
	mgr.OnCycleComplete(l.MachineIDs[1], true)
	if bms[0].IsBlockedByBuffer() {
		t.Fatal("stage 1 should be unblocked after space opened")
	}
	mustLedger(t, mgr)
}

func TestInterruptStopsRelease(t *testing.T) {
	mgr, l, _ := newTestLine(t, 3, 5)
	mgr.LoadJob("l1", JobSpec{OrderID: "o1", ProductID: "A", TargetQty: 50})
	mgr.StartJob("l1")
	mgr.OnCycleComplete(l.MachineIDs[0], true)
	if _, err := mgr.InterruptJob("l1"); err != nil {
		t.Fatal(err)
	}
	startedAtInterrupt := mustLedger(t, mgr).Started
	drive(t, mgr, l, 100, noScrap)
	if got := mustLedger(t, mgr).Started; got != startedAtInterrupt {
		t.Fatalf("started = %d after interrupt, want %d", got, startedAtInterrupt)
	}
	mgr.StartJob("l1")
	if len(busyStages(mgr)) == 0 {
		t.Fatal("resume should release a new part at stage 1")
	}
}

func TestShortFinishThenOtherProductScrapsLeftovers(t *testing.T) {
	mgr, l, _ := newTestLine(t, 3, 5)
	mgr.LoadJob("l1", JobSpec{OrderID: "o1", ProductID: "A", TargetQty: 100})
	mgr.StartJob("l1")
	for k := 0; k < 4; k++ {
		mgr.OnCycleComplete(l.MachineIDs[0], true)
	}
	if _, err := mgr.FinishJob("l1"); err != nil {
		t.Fatal(err)
	}
	left := mustLedger(t, mgr)
	leftover := left.InBuffers + left.InMachines
	if leftover == 0 {
		t.Fatal("test setup: expected parts left in the line")
	}
	if _, err := mgr.LoadJob("l1", JobSpec{OrderID: "o2", ProductID: "B", TargetQty: 10}); err != nil {
		t.Fatal(err)
	}
	led := mustLedger(t, mgr)
	if led.WIPScrapped != leftover || led.InBuffers+led.InMachines != 0 {
		t.Fatalf("ledger = %+v, want %d WIP scrapped and an empty line", led, leftover)
	}
}

func TestSameProductCarriesOver(t *testing.T) {
	tests := []struct {
		name   string
		target int
	}{
		{name: "next job needs more than the leftover", target: 10},
		{name: "next job needs fewer than the leftover", target: 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr, l, _ := newTestLine(t, 3, 5)
			mgr.LoadJob("l1", JobSpec{OrderID: "o1", ProductID: "A", TargetQty: 100})
			mgr.StartJob("l1")
			for k := 0; k < 4; k++ {
				mgr.OnCycleComplete(l.MachineIDs[0], true)
			}
			mgr.FinishJob("l1")
			left := mustLedger(t, mgr)
			leftover := left.InBuffers + left.InMachines

			js, err := mgr.LoadJob("l1", JobSpec{OrderID: "o2", ProductID: "A", TargetQty: tt.target})
			if err != nil {
				t.Fatal(err)
			}
			wantCarried := min(leftover, tt.target)
			led := mustLedger(t, mgr)
			if led.WIPCarriedOver != wantCarried || led.WIPScrapped != leftover-wantCarried || js.PartsInProcess != wantCarried {
				t.Fatalf("leftover %d: ledger = %+v job = %+v, want %d carried over", leftover, led, js, wantCarried)
			}
			mgr.StartJob("l1")
			drive(t, mgr, l, 1000, noScrap)
			if js := mustJob(t, mgr); js.Good != tt.target || js.State != JobTargetReached {
				t.Fatalf("job = %+v, want %d good", js, tt.target)
			}
		})
	}
}

func TestStallCheckIgnoresFaultAndRecoversStall(t *testing.T) {
	mgr, _, bms := newTestLine(t, 2, 3)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	for _, m := range bms {
		go m.Start(ctx)
	}
	mgr.LoadJob("l1", JobSpec{OrderID: "o1", ProductID: "A", TargetQty: 5})
	mgr.StartJob("l1")
	waitFor(t, func() bool { return bms[0].GetState() == machine.StateRunning })

	later := time.Now().Add(time.Hour)
	bms[0].SendCommand(machine.CommandFault)
	waitFor(t, func() bool { return bms[0].GetState() == machine.StateFault })
	if got := mgr.CheckStalls(later, time.Minute); len(got) != 0 {
		t.Fatalf("recovered %v during a fault, want none", got)
	}

	bms[0].SendCommand(machine.CommandReset) // FAULT → IDLE with a part, nobody restarts it
	waitFor(t, func() bool { return bms[0].GetState() == machine.StateIdle })
	if got := mgr.CheckStalls(later, time.Minute); len(got) != 1 {
		t.Fatalf("recovered %v, want l1", got)
	}
	waitFor(t, func() bool { return bms[0].GetState() == machine.StateRunning })
	mustLedger(t, mgr)
}

func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatal("condition not met within 2s")
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// TestRunningLineConservesParts runs real machine goroutines with scrap and
// faults through two jobs and checks the ledger.
func TestRunningLineConservesParts(t *testing.T) {
	if testing.Short() {
		t.Skip("runs machines for several seconds")
	}
	bus := event.New()
	eng := simulation.New(bus, 20, 0.1)
	li := config.LineInstanceConfig{InstanceID: "l1", TemplateName: "tpl"}
	machines := make(map[string]machine.Machine)
	for i := 1; i <= 4; i++ {
		mc := config.MachineInstanceConfig{
			ID: fmt.Sprintf("l1-s%d", i), Stage: i, LineInstanceID: "l1",
			CycleTimeMS: 400 + 100*i, ScrapRate: 0.05, BreakdownProbability: 0.03, AutoRestoreTimeMS: 2000,
		}
		li.Machines = append(li.Machines, mc)
		m := machine.NewBaseMachine(mc)
		eng.RegisterMachine(m)
		machines[mc.ID] = m
		if i > 1 {
			li.Buffers = append(li.Buffers, config.BufferInstanceConfig{
				ID: fmt.Sprintf("b%d", i-1), UpstreamMachineID: fmt.Sprintf("l1-s%d", i-1),
				DownstreamMachineID: mc.ID, Capacity: 3,
			})
		}
	}
	mgr := NewManager(bus)
	mgr.AddLine(InstantiateFromConfig(li), machines)
	mgr.SubscribeCycleEvents()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	eng.Start(ctx)
	defer eng.Stop()

	for k, target := range []int{15, 10} {
		if _, err := mgr.LoadJob("l1", JobSpec{OrderID: fmt.Sprintf("o%d", k), ProductID: "A", TargetQty: target}); err != nil {
			t.Fatal(err)
		}
		mgr.StartJob("l1")
		deadline := time.Now().Add(30 * time.Second)
		for mustJob(t, mgr).State != JobTargetReached {
			if time.Now().After(deadline) {
				t.Fatalf("job %d did not reach target: %+v ledger %+v", k, mustJob(t, mgr), mustLedger(t, mgr))
			}
			mgr.CheckStalls(time.Now(), 5*time.Second)
			time.Sleep(20 * time.Millisecond)
		}
		if js := mustJob(t, mgr); js.Good != target {
			t.Fatalf("job %d good = %d, want %d", k, js.Good, target)
		}
		mgr.FinishJob("l1")
	}
	led := mustLedger(t, mgr)
	if led.DeliveredGood != 25 {
		t.Fatalf("delivered %d, want 25 (ledger %+v)", led.DeliveredGood, led)
	}
}
