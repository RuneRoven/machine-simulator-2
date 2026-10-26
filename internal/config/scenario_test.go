package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeScenario(t *testing.T, dir, name, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, "scenarios"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "scenarios", name+".yaml"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func machineByStage(t *testing.T, li LineInstanceConfig, stage int) MachineInstanceConfig {
	t.Helper()
	for _, m := range li.Machines {
		if m.Stage == stage {
			return m
		}
	}
	t.Fatalf("line %s has no stage %d", li.InstanceID, stage)
	return MachineInstanceConfig{}
}

func TestScenarioAppliesStageAndBufferCharacters(t *testing.T) {
	dir := setupTestConfig(t)
	writeScenario(t, dir, "slow-press", `
name: slow-press
description: "press is the bottleneck"
lines:
  test-line:
    stages:
      1:
        cycle_time_factor: 1.5
        breakdown_factor: 4
        scrap_factor: 2
        auto_restore_time_ms: 150000
    buffers:
      - {from_stage: 1, to_stage: 2, capacity: 2}
`)
	t.Setenv("SIMULATOR_SCENARIO", "slow-press")

	cfg, err := Load(dir, "test-profile")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	for _, li := range cfg.Lines {
		m1 := machineByStage(t, li, 1)
		if m1.CycleTimeMS != 7500 || m1.BreakdownProbability != 0.004 || m1.ScrapRate != 0.04 || m1.AutoRestoreTimeMS != 150000 {
			t.Errorf("%s stage 1 = cycle %d, breakdown %v, scrap %v, restore %d; want 7500, 0.004, 0.04, 150000",
				li.InstanceID, m1.CycleTimeMS, m1.BreakdownProbability, m1.ScrapRate, m1.AutoRestoreTimeMS)
		}
		m2 := machineByStage(t, li, 2)
		if m2.CycleTimeMS != 3000 || m2.ScrapRate != 0.01 {
			t.Errorf("%s stage 2 changed: cycle %d scrap %v", li.InstanceID, m2.CycleTimeMS, m2.ScrapRate)
		}
		if got := li.Buffers[0].Capacity; got != 2 {
			t.Errorf("%s buffer 1-2 capacity %d, want 2", li.InstanceID, got)
		}
		// Recipe overrides of the stage scale too, so the character survives a product change.
		for _, r := range li.Recipes {
			if r.ProductID == "PRODUCT-A" {
				if ov := r.Overrides[1]; ov.CycleTimeMS == nil || *ov.CycleTimeMS != 6750 {
					t.Errorf("PRODUCT-A stage 1 recipe cycle = %v, want 6750 (4500 x 1.5)", ov.CycleTimeMS)
				}
			}
		}
	}
	// Deadlock threshold must exceed the longer restore time.
	if cfg.Global.DeadlockThresholdMS <= 150000 {
		t.Errorf("deadlock threshold %d, want > 150000", cfg.Global.DeadlockThresholdMS)
	}
	if cfg.Scenario.Name != "slow-press" {
		t.Errorf("Scenario.Name %q", cfg.Scenario.Name)
	}
}

func TestScenarioAppliesWithLineEnvVars(t *testing.T) {
	dir := setupTestConfig(t)
	writeScenario(t, dir, "s", "lines:\n  test-line:\n    stages:\n      2: {cycle_time_factor: 2}\n")
	t.Setenv("SIMULATOR_LINE_TEST_LINE", "1")
	t.Setenv("SIMULATOR_SCENARIO", "s")
	cfg, err := LoadFromLineEnvVars(dir)
	if err != nil {
		t.Fatalf("LoadFromLineEnvVars: %v", err)
	}
	if got := machineByStage(t, cfg.Lines[0], 2).CycleTimeMS; got != 6000 {
		t.Errorf("stage 2 cycle %d, want 6000", got)
	}
}

func TestScenarioRejectsBadReferences(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{name: "unknown template", body: "lines:\n  nope:\n    stages:\n      1: {cycle_time_factor: 2}\n", want: `unknown line template "nope"`},
		{name: "unknown stage", body: "lines:\n  test-line:\n    stages:\n      9: {cycle_time_factor: 2}\n", want: "no stage 9"},
		{name: "negative factor", body: "lines:\n  test-line:\n    stages:\n      1: {breakdown_factor: -1}\n", want: "breakdown_factor"},
		{name: "unknown buffer", body: "lines:\n  test-line:\n    buffers:\n      - {from_stage: 2, to_stage: 3, capacity: 1}\n", want: "buffer 2-3"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := setupTestConfig(t)
			writeScenario(t, dir, "bad", tt.body)
			t.Setenv("SIMULATOR_SCENARIO", "bad")
			_, err := Load(dir, "test-profile")
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("Load error %v, want one containing %q", err, tt.want)
			}
		})
	}
}

func TestScenarioMissingFile(t *testing.T) {
	dir := setupTestConfig(t)
	t.Setenv("SIMULATOR_SCENARIO", "does-not-exist")
	if _, err := Load(dir, "test-profile"); err == nil || !strings.Contains(err.Error(), "does-not-exist") {
		t.Errorf("Load error %v, want one naming the scenario", err)
	}
}

func TestNoScenarioLeavesTemplates(t *testing.T) {
	dir := setupTestConfig(t)
	cfg, err := Load(dir, "test-profile")
	if err != nil {
		t.Fatal(err)
	}
	if got := machineByStage(t, cfg.Lines[0], 1).CycleTimeMS; got != 5000 {
		t.Errorf("stage 1 cycle %d, want template 5000", got)
	}
}
