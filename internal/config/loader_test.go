package config

import (
	"os"
	"path/filepath"
	"testing"
)

func setupTestConfig(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	// global.yaml
	os.WriteFile(filepath.Join(dir, "global.yaml"), []byte(`
time_scale: 2.0
random_factor: 0.15
tick_rate_ms: 100
log_level: debug
api:
  host: "0.0.0.0"
  port: 8081
erp:
  mode: auto
  auto_order_interval_ms: 30000
  max_backlog: 5
mes:
  dispatch_interval_ms: 1000
  dispatch_strategy: round-robin
`), 0644)

	// Line template
	os.MkdirAll(filepath.Join(dir, "lines", "test-domain"), 0755)
	os.WriteFile(filepath.Join(dir, "lines", "test-domain", "test-line.yaml"), []byte(`
name: test-line
domain: test-domain
description: "Test production line"
machines:
  - stage: 1
    type: metal_forming
    role: "Press"
    defaults:
      cycle_time_ms: 5000
      scrap_rate: 0.02
      breakdown_probability: 0.001
  - stage: 2
    type: spot_welder
    role: "Welder"
    defaults:
      cycle_time_ms: 3000
      scrap_rate: 0.01
buffers:
  default_capacity: 10
  overrides:
    - from_stage: 1
      to_stage: 2
      capacity: 15
recipes:
  - product_id: "PRODUCT-A"
    description: "Product A"
    planned_qty_range: [50, 200]
    overrides:
      1:
        cycle_time_ms: 4500
  - product_id: "PRODUCT-B"
    description: "Product B"
    planned_qty_range: [30, 100]
`), 0644)

	// Profile
	os.MkdirAll(filepath.Join(dir, "profiles"), 0755)
	os.WriteFile(filepath.Join(dir, "profiles", "test-profile.yaml"), []byte(`
name: test-profile
description: "Test profile"
lines:
  - template: test-line
    instances: 2
overrides:
  global:
    time_scale: 5.0
`), 0644)

	// Adapter configs
	os.MkdirAll(filepath.Join(dir, "adapters"), 0755)
	os.WriteFile(filepath.Join(dir, "adapters", "opcua.yaml"), []byte(`
enabled: true
base_port: 4840
update_rate_ms: 100
max_sessions: 50
hostname: localhost
`), 0644)
	os.WriteFile(filepath.Join(dir, "adapters", "modbus.yaml"), []byte(`
enabled: false
port: 502
`), 0644)

	return dir
}

func TestLoadValidConfig(t *testing.T) {
	dir := setupTestConfig(t)

	cfg, err := Load(dir, "test-profile")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Profile override applied
	if cfg.Global.TimeScale != 5.0 {
		t.Errorf("expected time_scale 5.0, got %f", cfg.Global.TimeScale)
	}

	// 2 instances of test-line
	if len(cfg.Lines) != 2 {
		t.Fatalf("expected 2 line instances, got %d", len(cfg.Lines))
	}

	li1 := cfg.Lines[0]
	if li1.InstanceID != "test-line-1" {
		t.Errorf("expected instance ID test-line-1, got %s", li1.InstanceID)
	}
	if len(li1.Machines) != 2 {
		t.Errorf("expected 2 machines, got %d", len(li1.Machines))
	}
	if li1.Machines[0].ID != "test-line-1-stage-1" {
		t.Errorf("expected machine ID test-line-1-stage-1, got %s", li1.Machines[0].ID)
	}

	// Buffer override applied
	if len(li1.Buffers) != 1 {
		t.Fatalf("expected 1 buffer, got %d", len(li1.Buffers))
	}
	if li1.Buffers[0].Capacity != 15 {
		t.Errorf("expected buffer capacity 15 (overridden), got %d", li1.Buffers[0].Capacity)
	}

	// Recipes
	if len(li1.Recipes) != 2 {
		t.Errorf("expected 2 recipes, got %d", len(li1.Recipes))
	}

	// Adapter configs
	if !cfg.Adapters.OPCUA.Enabled {
		t.Error("expected OPC-UA enabled")
	}
	if cfg.Adapters.Modbus.Enabled {
		t.Error("expected Modbus disabled")
	}
}

func TestLoadMissingProfile(t *testing.T) {
	dir := setupTestConfig(t)

	_, err := Load(dir, "nonexistent")
	if err == nil {
		t.Fatal("expected error for missing profile")
	}
}

func TestLoadUnknownTemplate(t *testing.T) {
	dir := setupTestConfig(t)

	os.WriteFile(filepath.Join(dir, "profiles", "bad-profile.yaml"), []byte(`
name: bad-profile
lines:
  - template: nonexistent-line
    instances: 1
`), 0644)

	_, err := Load(dir, "bad-profile")
	if err == nil {
		t.Fatal("expected error for unknown template")
	}
}

func TestValidationMinRecipes(t *testing.T) {
	dir := t.TempDir()

	os.MkdirAll(filepath.Join(dir, "lines", "domain"), 0755)
	os.WriteFile(filepath.Join(dir, "lines", "domain", "bad-line.yaml"), []byte(`
name: bad-line
domain: domain
machines:
  - stage: 1
    type: test
    role: "Test"
    defaults:
      cycle_time_ms: 1000
recipes:
  - product_id: "ONLY-ONE"
    description: "Only one recipe"
    planned_qty_range: [10, 50]
`), 0644)

	os.MkdirAll(filepath.Join(dir, "profiles"), 0755)
	os.WriteFile(filepath.Join(dir, "profiles", "p.yaml"), []byte(`
name: p
lines:
  - template: bad-line
    instances: 1
`), 0644)

	_, err := Load(dir, "p")
	if err == nil {
		t.Fatal("expected validation error for < 2 recipes")
	}
}

func TestValidationInvalidScrapRate(t *testing.T) {
	cfg := &Config{
		Global:    DefaultGlobalConfig(),
		Templates: map[string]*LineTemplateConfig{},
		Lines: []LineInstanceConfig{
			{
				InstanceID:   "line-1",
				TemplateName: "test",
				Machines: []MachineInstanceConfig{
					{ID: "m1", ScrapRate: 1.5},
				},
			},
		},
	}

	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected validation error for scrap_rate > 1.0")
	}
}

func TestEnvOverrides(t *testing.T) {
	dir := setupTestConfig(t)

	t.Setenv("SIMULATOR_TIME_SCALE", "99.0")

	cfg, err := Load(dir, "test-profile")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Env should override profile override
	if cfg.Global.TimeScale != 99.0 {
		t.Errorf("expected time_scale 99.0 from env, got %f", cfg.Global.TimeScale)
	}
}

func TestEmptyProfile(t *testing.T) {
	dir := setupTestConfig(t)

	os.WriteFile(filepath.Join(dir, "profiles", "empty.yaml"), []byte(`
name: empty
lines: []
`), 0644)

	_, err := Load(dir, "empty")
	if err == nil {
		t.Fatal("expected error for empty profile")
	}
}

func TestZeroInstances(t *testing.T) {
	dir := setupTestConfig(t)

	os.WriteFile(filepath.Join(dir, "profiles", "zero.yaml"), []byte(`
name: zero
lines:
  - template: test-line
    instances: 0
`), 0644)

	_, err := Load(dir, "zero")
	if err == nil {
		t.Fatal("expected error for profile with 0 instances")
	}
}

func TestTemplateToEnvVar(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"automotive-welding", "SIMULATOR_LINE_AUTOMOTIVE_WELDING"},
		{"electronics-smt", "SIMULATOR_LINE_ELECTRONICS_SMT"},
		{"food-beverage-filling", "SIMULATOR_LINE_FOOD_BEVERAGE_FILLING"},
		{"test-line", "SIMULATOR_LINE_TEST_LINE"},
	}
	for _, tc := range tests {
		got := templateToEnvVar(tc.input)
		if got != tc.expected {
			t.Errorf("templateToEnvVar(%q) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}

func TestLoadFromLineEnvVars(t *testing.T) {
	dir := setupTestConfig(t)

	t.Setenv("SIMULATOR_LINE_TEST_LINE", "2")

	cfg, err := LoadFromLineEnvVars(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected non-nil config when env var is set")
	}

	if len(cfg.Lines) != 2 {
		t.Fatalf("expected 2 line instances, got %d", len(cfg.Lines))
	}
	if cfg.Lines[0].InstanceID != "test-line-1" {
		t.Errorf("expected instance ID test-line-1, got %s", cfg.Lines[0].InstanceID)
	}
	if cfg.Lines[1].InstanceID != "test-line-2" {
		t.Errorf("expected instance ID test-line-2, got %s", cfg.Lines[1].InstanceID)
	}
	if cfg.Profile.Name != "env-vars" {
		t.Errorf("expected profile name 'env-vars', got %q", cfg.Profile.Name)
	}
}

func TestLoadFromLineEnvVarsNoneSet(t *testing.T) {
	dir := setupTestConfig(t)

	cfg, err := LoadFromLineEnvVars(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg != nil {
		t.Fatal("expected nil config when no env vars are set")
	}
}

func TestLoadFromLineEnvVarsZeroSkipped(t *testing.T) {
	dir := setupTestConfig(t)

	t.Setenv("SIMULATOR_LINE_TEST_LINE", "0")

	cfg, err := LoadFromLineEnvVars(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg != nil {
		t.Fatal("expected nil config when env var is set to 0")
	}
}
