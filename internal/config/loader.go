package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
)

// Load assembles the full configuration from:
// 1. Built-in defaults
// 2. config/global.yaml
// 3. Profile YAML (determines active lines)
// 4. Line template YAMLs
// 5. Adapter config YAMLs
// 6. Environment variable overrides
//
// configDir is the base config directory (e.g., "config").
func Load(configDir, profileName string) (*Config, error) {
	cfg := &Config{
		Global:    DefaultGlobalConfig(),
		Templates: make(map[string]*LineTemplateConfig),
		Adapters: AdaptersConfig{
			OPCUA:  DefaultOPCUAAdapterConfig(),
			Modbus: DefaultModbusAdapterConfig(),
		},
	}

	// 1. Load global.yaml (optional — defaults are used if missing)
	globalPath := filepath.Join(configDir, "global.yaml")
	if err := loadYAMLFile(globalPath, &cfg.Global); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("loading global config: %w", err)
	}

	// 2. Load all line templates from config/lines/
	linesDir := filepath.Join(configDir, "lines")
	if err := loadLineTemplates(linesDir, cfg.Templates); err != nil {
		return nil, fmt.Errorf("loading line templates: %w", err)
	}

	// 3. Load profile
	profilePath := filepath.Join(configDir, "profiles", profileName+".yaml")
	if err := loadYAMLFile(profilePath, &cfg.Profile); err != nil {
		return nil, fmt.Errorf("loading profile %q: %w", profileName, err)
	}

	// 4. Apply profile overrides to global config
	applyProfileOverrides(&cfg.Global, cfg.Profile.Overrides)

	// 5. Load adapter configs
	opcuaPath := filepath.Join(configDir, "adapters", "opcua.yaml")
	if err := loadYAMLFile(opcuaPath, &cfg.Adapters.OPCUA); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("loading OPC-UA adapter config: %w", err)
	}
	modbusPath := filepath.Join(configDir, "adapters", "modbus.yaml")
	if err := loadYAMLFile(modbusPath, &cfg.Adapters.Modbus); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("loading Modbus adapter config: %w", err)
	}

	// 6. Apply environment variable overrides
	applyEnvOverrides(&cfg.Global, &cfg.Adapters)

	// 7. Instantiate lines from profile
	if err := instantiateLines(cfg); err != nil {
		return nil, fmt.Errorf("instantiating lines: %w", err)
	}

	// 8. Validate
	if err := Validate(cfg); err != nil {
		return nil, fmt.Errorf("config validation: %w", err)
	}

	return cfg, nil
}

func loadYAMLFile(path string, target interface{}) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(data, target)
}

func loadLineTemplates(linesDir string, templates map[string]*LineTemplateConfig) error {
	if _, err := os.Stat(linesDir); os.IsNotExist(err) {
		return nil // no lines directory is valid (though validation will fail later)
	}

	return filepath.Walk(linesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, ".yaml") {
			return nil
		}

		var tmpl LineTemplateConfig
		if err := loadYAMLFile(path, &tmpl); err != nil {
			return fmt.Errorf("loading template %s: %w", path, err)
		}

		// Derive domain from parent directory name if not set
		if tmpl.Domain == "" {
			tmpl.Domain = filepath.Base(filepath.Dir(path))
		}

		if tmpl.Name == "" {
			return fmt.Errorf("template at %s has no name", path)
		}

		if _, exists := templates[tmpl.Name]; exists {
			return fmt.Errorf("duplicate template name %q (found in %s)", tmpl.Name, path)
		}

		templates[tmpl.Name] = &tmpl
		return nil
	})
}

func applyProfileOverrides(global *GlobalConfig, overrides ProfileOverrides) {
	if overrides.Global == nil {
		return
	}
	if v, ok := overrides.Global["time_scale"]; ok {
		if f, ok := toFloat64(v); ok {
			global.TimeScale = f
		}
	}
	if v, ok := overrides.Global["random_factor"]; ok {
		if f, ok := toFloat64(v); ok {
			global.RandomFactor = f
		}
	}
}

func applyEnvOverrides(global *GlobalConfig, adapters *AdaptersConfig) {
	if v := os.Getenv("SIMULATOR_TIME_SCALE"); v != "" {
		if f, ok := parseFloat(v); ok {
			global.TimeScale = f
		}
	}
	if v := os.Getenv("SIMULATOR_RANDOM_FACTOR"); v != "" {
		if f, ok := parseFloat(v); ok {
			global.RandomFactor = f
		}
	}
	if v := os.Getenv("SIMULATOR_API_HOST"); v != "" {
		global.API.Host = v
	}
	if v := os.Getenv("SIMULATOR_API_PORT"); v != "" {
		if i, ok := parseInt(v); ok {
			global.API.Port = i
		}
	}
	if v := os.Getenv("SIMULATOR_ERP_MODE"); v != "" {
		global.ERP.Mode = v
	}
	if v := os.Getenv("SIMULATOR_ERP_AUTO_ORDER_INTERVAL_MS"); v != "" {
		if i, ok := parseInt(v); ok {
			global.ERP.AutoOrderIntervalMS = i
		}
	}
	if v := os.Getenv("SIMULATOR_MES_DISPATCH_STRATEGY"); v != "" {
		global.MES.DispatchStrategy = v
	}
	if v := os.Getenv("SIMULATOR_OPCUA_ENABLED"); v != "" {
		adapters.OPCUA.Enabled = v == "true" || v == "1"
	}
	if v := os.Getenv("SIMULATOR_OPCUA_HOSTNAME"); v != "" {
		adapters.OPCUA.Hostname = v
	}
	if v := os.Getenv("SIMULATOR_OPCUA_BASE_PORT"); v != "" {
		if i, ok := parseInt(v); ok {
			adapters.OPCUA.BasePort = i
		}
	}
	if v := os.Getenv("SIMULATOR_MODBUS_ENABLED"); v != "" {
		adapters.Modbus.Enabled = v == "true" || v == "1"
	}
}

func instantiateLines(cfg *Config) error {
	for _, entry := range cfg.Profile.Lines {
		if entry.Instances <= 0 {
			continue
		}

		tmpl, ok := cfg.Templates[entry.Template]
		if !ok {
			available := make([]string, 0, len(cfg.Templates))
			for name := range cfg.Templates {
				available = append(available, name)
			}
			return fmt.Errorf("profile references unknown template %q; available: %v", entry.Template, available)
		}

		for i := 1; i <= entry.Instances; i++ {
			instanceID := fmt.Sprintf("%s-%d", tmpl.Name, i)
			li := LineInstanceConfig{
				InstanceID:   instanceID,
				TemplateName: tmpl.Name,
				Domain:       tmpl.Domain,
				Recipes:      tmpl.Recipes,
			}

			defaults := DefaultMachineDefaults()

			for _, stage := range tmpl.Machines {
				machineID := fmt.Sprintf("%s-stage-%d", instanceID, stage.Stage)

				cycleTime := stage.Defaults.CycleTimeMS
				if cycleTime == 0 {
					cycleTime = defaults.CycleTimeMS
				}
				scrapRate := stage.Defaults.ScrapRate
				breakdownProb := stage.Defaults.BreakdownProbability
				autoRestore := stage.Defaults.AutoRestoreTimeMS
				if autoRestore == 0 {
					autoRestore = defaults.AutoRestoreTimeMS
				}

				li.Machines = append(li.Machines, MachineInstanceConfig{
					ID:                   machineID,
					Type:                 stage.Type,
					Role:                 stage.Role,
					LineInstanceID:       instanceID,
					Stage:                stage.Stage,
					CycleTimeMS:          cycleTime,
					ScrapRate:            scrapRate,
					BreakdownProbability: breakdownProb,
					AutoRestoreTimeMS:    autoRestore,
				})
			}

			// Create buffers between consecutive stages
			for j := 0; j < len(li.Machines)-1; j++ {
				upstream := li.Machines[j]
				downstream := li.Machines[j+1]
				cap := tmpl.Buffers.DefaultCapacity
				if cap == 0 {
					cap = 10
				}
				// Check for overrides
				for _, ov := range tmpl.Buffers.Overrides {
					if ov.FromStage == upstream.Stage && ov.ToStage == downstream.Stage {
						cap = ov.Capacity
						break
					}
				}

				bufferID := fmt.Sprintf("%s-buffer-%d-%d", instanceID, upstream.Stage, downstream.Stage)
				li.Buffers = append(li.Buffers, BufferInstanceConfig{
					ID:                  bufferID,
					UpstreamMachineID:   upstream.ID,
					DownstreamMachineID: downstream.ID,
					Capacity:            cap,
				})
			}

			cfg.Lines = append(cfg.Lines, li)
		}
	}
	return nil
}

func toFloat64(v interface{}) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case int:
		return float64(n), true
	case float32:
		return float64(n), true
	default:
		return 0, false
	}
}

func parseFloat(s string) (float64, bool) {
	var f float64
	_, err := fmt.Sscanf(s, "%f", &f)
	return f, err == nil
}

func parseInt(s string) (int, bool) {
	var i int
	_, err := fmt.Sscanf(s, "%d", &i)
	return i, err == nil
}

// templateToEnvVar converts a line template name to its environment variable name.
// e.g., "automotive-welding" -> "SIMULATOR_LINE_AUTOMOTIVE_WELDING"
func templateToEnvVar(templateName string) string {
	upper := strings.ToUpper(templateName)
	return "SIMULATOR_LINE_" + strings.ReplaceAll(upper, "-", "_")
}

// LoadFromLineEnvVars builds a Config from per-line environment variables.
// Each SIMULATOR_LINE_<NAME> env var specifies the number of instances for that template.
// Returns (nil, nil) if no SIMULATOR_LINE_* vars are set with value > 0, signaling
// the caller should fall back to profile-based loading.
func LoadFromLineEnvVars(configDir string) (*Config, error) {
	cfg := &Config{
		Global:    DefaultGlobalConfig(),
		Templates: make(map[string]*LineTemplateConfig),
		Adapters: AdaptersConfig{
			OPCUA:  DefaultOPCUAAdapterConfig(),
			Modbus: DefaultModbusAdapterConfig(),
		},
	}

	// 1. Load global.yaml
	globalPath := filepath.Join(configDir, "global.yaml")
	if err := loadYAMLFile(globalPath, &cfg.Global); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("loading global config: %w", err)
	}

	// 2. Load all line templates
	linesDir := filepath.Join(configDir, "lines")
	if err := loadLineTemplates(linesDir, cfg.Templates); err != nil {
		return nil, fmt.Errorf("loading line templates: %w", err)
	}

	// 3. Build profile lines from env vars
	var lines []ProfileLineEntry
	for name := range cfg.Templates {
		envVar := templateToEnvVar(name)
		v := os.Getenv(envVar)
		if v == "" {
			continue
		}
		count, ok := parseInt(v)
		if !ok || count <= 0 {
			continue
		}
		log.Info().Str("env", envVar).Int("instances", count).Str("template", name).Msg("Line selected via env var")
		lines = append(lines, ProfileLineEntry{
			Template:  name,
			Instances: count,
		})
	}

	// No lines selected — signal caller to fall back to profile
	if len(lines) == 0 {
		return nil, nil
	}

	// Sort lines alphabetically by template name for deterministic port assignment
	sort.Slice(lines, func(i, j int) bool {
		return lines[i].Template < lines[j].Template
	})

	cfg.Profile = ProfileConfig{
		Name:  "env-vars",
		Lines: lines,
	}

	// 4. Load adapter configs
	opcuaPath := filepath.Join(configDir, "adapters", "opcua.yaml")
	if err := loadYAMLFile(opcuaPath, &cfg.Adapters.OPCUA); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("loading OPC-UA adapter config: %w", err)
	}
	modbusPath := filepath.Join(configDir, "adapters", "modbus.yaml")
	if err := loadYAMLFile(modbusPath, &cfg.Adapters.Modbus); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("loading Modbus adapter config: %w", err)
	}

	// 5. Apply environment variable overrides
	applyEnvOverrides(&cfg.Global, &cfg.Adapters)

	// 6. Instantiate lines
	if err := instantiateLines(cfg); err != nil {
		return nil, fmt.Errorf("instantiating lines: %w", err)
	}

	// 7. Validate
	if err := Validate(cfg); err != nil {
		return nil, fmt.Errorf("config validation: %w", err)
	}

	return cfg, nil
}

// Validate checks the assembled configuration for consistency.
func Validate(cfg *Config) error {
	// Profile must have at least one active line
	if len(cfg.Lines) == 0 {
		return fmt.Errorf("no lines configured — profile has no line entries or all have instances=0")
	}

	// Check for duplicate machine IDs
	machineIDs := make(map[string]string) // machineID -> lineInstanceID
	for _, li := range cfg.Lines {
		for _, m := range li.Machines {
			if existing, dup := machineIDs[m.ID]; dup {
				return fmt.Errorf("duplicate machine ID %q in line instances %q and %q", m.ID, existing, li.InstanceID)
			}
			machineIDs[m.ID] = li.InstanceID
		}
	}

	// Validate each line template
	for name, tmpl := range cfg.Templates {
		if len(tmpl.Recipes) < 2 {
			return fmt.Errorf("template %q has %d recipes (minimum 2 required)", name, len(tmpl.Recipes))
		}
		for _, stage := range tmpl.Machines {
			if stage.Type == "" {
				return fmt.Errorf("template %q stage %d has empty machine type", name, stage.Stage)
			}
		}
	}

	// Validate value ranges
	if cfg.Global.TimeScale < 0 {
		return fmt.Errorf("time_scale must be >= 0, got %f", cfg.Global.TimeScale)
	}
	if cfg.Global.RandomFactor < 0 || cfg.Global.RandomFactor > 1.0 {
		return fmt.Errorf("random_factor must be between 0 and 1, got %f", cfg.Global.RandomFactor)
	}

	for _, li := range cfg.Lines {
		for _, m := range li.Machines {
			if m.ScrapRate < 0 || m.ScrapRate > 1.0 {
				return fmt.Errorf("machine %q scrap_rate must be between 0 and 1, got %f", m.ID, m.ScrapRate)
			}
			if m.BreakdownProbability < 0 || m.BreakdownProbability > 1.0 {
				return fmt.Errorf("machine %q breakdown_probability must be between 0 and 1, got %f", m.ID, m.BreakdownProbability)
			}
		}
	}

	return nil
}
