package opcua

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/gopcua/opcua/ua"

	"machine-simulator-2/internal/config"
	"machine-simulator-2/internal/event"
	"machine-simulator-2/internal/line"
	"machine-simulator-2/internal/machine"
)

func newLineServerUnderTest(t *testing.T) (*lineServer, *line.Manager) {
	t.Helper()
	li := config.LineInstanceConfig{
		InstanceID: "l1", TemplateName: "tpl",
		Recipes: []config.RecipeConfig{{ProductID: "A"}, {ProductID: "B"}},
	}
	machines := make(map[string]machine.Machine)
	for i := 1; i <= 2; i++ {
		mc := config.MachineInstanceConfig{ID: fmt.Sprintf("l1-s%d", i), Stage: i, CycleTimeMS: 100, LineInstanceID: "l1"}
		li.Machines = append(li.Machines, mc)
		machines[mc.ID] = machine.NewBaseMachine(mc)
	}
	li.Buffers = []config.BufferInstanceConfig{{ID: "b1", UpstreamMachineID: "l1-s1", DownstreamMachineID: "l1-s2", Capacity: 3}}
	mgr := line.NewManager(event.New())
	mgr.AddLine(line.InstantiateFromConfig(li), machines)

	// Port 0: the OS picks a free port.
	ls := NewLineServers(config.OPCUAAdapterConfig{Hostname: "localhost", LineBasePort: 0, UpdateRateMS: 50}, mgr)
	ls.Register("l1")
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	ls.Start(ctx)
	s := ls.servers[0]
	if !s.started.Load() {
		t.Fatal("line server did not start")
	}
	return s, mgr
}

// clientWrite goes through the namespace exactly as the server's Write
// service does for a client request.
func clientWrite(s *lineServer, nodeID uint32, v interface{}) ua.StatusCode {
	return s.ns.SetAttribute(ua.NewNumericNodeID(s.ns.ID(), nodeID), ua.AttributeIDValue, &ua.DataValue{
		EncodingMask: ua.DataValueValue,
		Value:        ua.MustVariant(v),
	})
}

func eventually(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func TestLineServerLoadAndStartByWrites(t *testing.T) {
	s, mgr := newLineServerUnderTest(t)

	for _, w := range []struct {
		node uint32
		v    interface{}
	}{
		{1, "4711"}, {2, "0010"}, {3, "A"}, {4, int32(25)}, {5, CommandLoad}, {5, CommandStart},
	} {
		if st := clientWrite(s, w.node, w.v); st != ua.StatusOK {
			t.Fatalf("write node %d: %v", w.node, st)
		}
	}
	eventually(t, "job RUNNING", func() bool {
		js, _ := mgr.Job("l1")
		return js.State == line.JobRunning
	})
	js, _ := mgr.Job("l1")
	if js.OrderID != "4711" || js.OperationID != "0010" || js.ProductID != "A" || js.TargetQty != 25 {
		t.Fatalf("job = %+v, want order 4711 op 0010 product A target 25", js)
	}
	// The command node keeps the written value so a read-back handshake passes.
	if got := s.setpoint(nodeJobCommand); got != CommandStart {
		t.Fatalf("job_command reads %v, want %d", got, CommandStart)
	}
	eventually(t, "result OK", func() bool { return s.resultNow() == "OK" })
}

func TestLineServerRejectsBadCommandsAndReadOnlyWrites(t *testing.T) {
	s, mgr := newLineServerUnderTest(t)

	clientWrite(s, 5, CommandStart) // nothing loaded
	eventually(t, "rejection", func() bool {
		r := s.resultNow()
		return len(r) > 9 && r[:9] == "rejected:"
	})

	clientWrite(s, 1, "o1")
	clientWrite(s, 3, "UNKNOWN")
	clientWrite(s, 5, CommandLoad)
	eventually(t, "unknown product rejection", func() bool {
		r := s.resultNow()
		return r != "" && r != "OK" && r[:9] == "rejected:"
	})
	if js, _ := mgr.Job("l1"); js.State != line.JobNone {
		t.Fatalf("state = %s after rejected load, want NO_JOB", js.State)
	}

	if st := clientWrite(s, 8, int32(99)); st != ua.StatusBadUserAccessDenied {
		t.Fatalf("write to job_good = %v, want BadUserAccessDenied", st)
	}
}

func TestLineServerRepeatedCommandIsHandled(t *testing.T) {
	s, mgr := newLineServerUnderTest(t)
	clientWrite(s, 1, "o1")
	clientWrite(s, 3, "A")
	clientWrite(s, 5, CommandLoad)
	eventually(t, "loaded", func() bool { js, _ := mgr.Job("l1"); return js.State == line.JobLoaded })

	// Same value again with new parameters: the load must run a second time.
	clientWrite(s, 1, "o2")
	clientWrite(s, 5, CommandLoad)
	eventually(t, "reloaded", func() bool { js, _ := mgr.Job("l1"); return js.OrderID == "o2" })
}
