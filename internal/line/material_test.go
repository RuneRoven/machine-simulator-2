package line

import (
	"errors"
	"math"
	"testing"
)

func TestJobWithoutComponentsNeedsNoMaterial(t *testing.T) {
	mgr, l, _ := newTestLine(t, 2, 2)
	mustLoadStart(t, mgr, JobSpec{OrderID: "o1", ProductID: "A", TargetQty: 5})
	drive(t, mgr, l, 1000, noScrap)
	if js := mustJob(t, mgr); js.State != JobTargetReached || js.MaterialMissing != "" {
		t.Fatalf("job %+v, want TARGET_REACHED without missing material", js)
	}
}

func TestStageOneConsumesComponentsAndStarvesWhenShort(t *testing.T) {
	mgr, l, _ := newTestLine(t, 2, 2)
	if _, err := mgr.AddMaterial("l1", "GLASS", 3); err != nil {
		t.Fatal(err)
	}
	if _, err := mgr.AddMaterial("l1", "PROFILE", 100); err != nil {
		t.Fatal(err)
	}
	mustLoadStart(t, mgr, JobSpec{OrderID: "o1", ProductID: "A", TargetQty: 10,
		Components: []Component{{Material: "GLASS", PerPart: 1}, {Material: "PROFILE", PerPart: 4.4}}})
	drive(t, mgr, l, 1000, noScrap)

	js := mustJob(t, mgr)
	if js.Good != 3 || js.State != JobRunning || js.MaterialMissing != "GLASS" {
		t.Fatalf("after 3 glasses: job %+v, want 3 good, RUNNING, waiting for GLASS", js)
	}
	if got := js.Material["PROFILE"]; math.Abs(got-(100-3*4.4)) > 1e-9 {
		t.Errorf("profile left %v, want %v", got, 100-3*4.4)
	}
	if js.Material["GLASS"] != 0 {
		t.Errorf("glass left %v, want 0", js.Material["GLASS"])
	}

	// Staging brings more: the line runs on by itself.
	if _, err := mgr.AddMaterial("l1", "GLASS", 7); err != nil {
		t.Fatal(err)
	}
	if js := mustJob(t, mgr); js.MaterialMissing != "" {
		t.Errorf("after staging: still missing %q", js.MaterialMissing)
	}
	drive(t, mgr, l, 1000, noScrap)
	if js := mustJob(t, mgr); js.State != JobTargetReached || js.Good != 10 {
		t.Fatalf("job %+v, want TARGET_REACHED with 10 good", js)
	}
	mustLedger(t, mgr)
}

func TestScrapConsumesMaterialToo(t *testing.T) {
	mgr, l, _ := newTestLine(t, 2, 2)
	if _, err := mgr.AddMaterial("l1", "SHEET", 100); err != nil {
		t.Fatal(err)
	}
	mustLoadStart(t, mgr, JobSpec{OrderID: "o1", ProductID: "A", TargetQty: 10, Components: []Component{{Material: "SHEET", PerPart: 0.5}}})
	drive(t, mgr, l, 1000, func(step, stage int) bool { return stage == 1 && step%5 == 0 })
	js := mustJob(t, mgr)
	if got, want := js.Material["SHEET"], 100-0.5*float64(js.Started); math.Abs(got-want) > 1e-9 {
		t.Errorf("sheet left %v after %d parts started, want %v", got, js.Started, want)
	}
	if js.Started <= 10 {
		t.Errorf("started %d, want more than 10 (scrap was replaced)", js.Started)
	}
}

func TestMaterialStaysOnTheLineBetweenJobs(t *testing.T) {
	mgr, l, _ := newTestLine(t, 2, 2)
	if _, err := mgr.AddMaterial("l1", "GLASS", 10); err != nil {
		t.Fatal(err)
	}
	mustLoadStart(t, mgr, JobSpec{OrderID: "o1", ProductID: "A", TargetQty: 4, Components: []Component{{Material: "GLASS", PerPart: 1}}})
	drive(t, mgr, l, 1000, noScrap)
	if _, err := mgr.FinishJob("l1"); err != nil {
		t.Fatal(err)
	}
	levels, _ := mgr.Material("l1")
	if levels["GLASS"] != 6 {
		t.Errorf("glass after the first job: %v, want 6", levels["GLASS"])
	}
}

func TestAddMaterialValidates(t *testing.T) {
	mgr, _, _ := newTestLine(t, 1, 1)
	if _, err := mgr.AddMaterial("l1", "", 5); !errors.Is(err, ErrInvalidJob) {
		t.Errorf("empty material: err %v, want ErrInvalidJob", err)
	}
	if _, err := mgr.AddMaterial("l1", "X", 0); !errors.Is(err, ErrInvalidJob) {
		t.Errorf("zero quantity: err %v, want ErrInvalidJob", err)
	}
	if _, err := mgr.AddMaterial("nope", "X", 1); !errors.Is(err, ErrUnknownLine) {
		t.Errorf("unknown line: err %v, want ErrUnknownLine", err)
	}
	if _, err := mgr.LoadJob("l1", JobSpec{OrderID: "o", ProductID: "A", Components: []Component{{Material: "X", PerPart: -1}}}); !errors.Is(err, ErrInvalidJob) {
		t.Errorf("negative per-part quantity: err %v, want ErrInvalidJob", err)
	}
}

func mustLoadStart(t *testing.T, mgr *Manager, spec JobSpec) {
	t.Helper()
	if _, err := mgr.LoadJob("l1", spec); err != nil {
		t.Fatal(err)
	}
	if _, err := mgr.StartJob("l1"); err != nil {
		t.Fatal(err)
	}
}
