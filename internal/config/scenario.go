package config

import (
	"errors"
	"fmt"
	"math"
	"path/filepath"
	"sort"

	"github.com/rs/zerolog/log"
)

// ScenarioConfig gives line templates a character for a demo: a bottleneck
// stage, a buffer that runs full, a machine that breaks down a lot. It lives in
// config/scenarios/<name>.yaml and is selected with global.scenario or
// SIMULATOR_SCENARIO. It changes templates, so every instance of a template
// gets the same character.
type ScenarioConfig struct {
	Name        string                        `yaml:"name"`
	Description string                        `yaml:"description"`
	Lines       map[string]ScenarioLineConfig `yaml:"lines"` // keyed by template name
}

// ScenarioLineConfig changes stages and buffers of one line template.
type ScenarioLineConfig struct {
	Stages  map[int]ScenarioStageConfig `yaml:"stages"`
	Buffers []BufferOverrideConfig      `yaml:"buffers"`
}

// ScenarioStageConfig changes one stage. Factors scale the template default and
// every recipe override of the stage, so the character survives a product
// change; 0 leaves the value alone.
type ScenarioStageConfig struct {
	CycleTimeFactor   float64 `yaml:"cycle_time_factor"`
	BreakdownFactor   float64 `yaml:"breakdown_factor"`
	ScrapFactor       float64 `yaml:"scrap_factor"`
	AutoRestoreTimeMS int     `yaml:"auto_restore_time_ms"`
	Note              string  `yaml:"note"`
}

// applyScenario loads the configured scenario and applies it to cfg.Templates.
func applyScenario(configDir string, cfg *Config) error {
	name := cfg.Global.Scenario
	if name == "" {
		return nil
	}
	var sc ScenarioConfig
	if err := loadYAMLFile(filepath.Join(configDir, "scenarios", name+".yaml"), &sc); err != nil {
		return fmt.Errorf("loading scenario %q: %w", name, err)
	}
	if sc.Name == "" {
		sc.Name = name
	}
	if err := sc.applyTo(cfg.Templates); err != nil {
		return fmt.Errorf("scenario %q: %w", name, err)
	}
	cfg.Scenario = sc
	log.Info().Str("scenario", sc.Name).Int("line_templates", len(sc.Lines)).Msg("Scenario applied to line templates")
	return nil
}

func (sc ScenarioConfig) applyTo(templates map[string]*LineTemplateConfig) error {
	names := make([]string, 0, len(sc.Lines))
	for n := range sc.Lines {
		names = append(names, n)
	}
	sort.Strings(names) // stable error order
	var errs []error
	for _, tmplName := range names {
		lc := sc.Lines[tmplName]
		tmpl, ok := templates[tmplName]
		if !ok {
			errs = append(errs, fmt.Errorf("unknown line template %q", tmplName))
			continue
		}
		for stage, st := range lc.Stages {
			if err := applyStage(tmpl, stage, st); err != nil {
				errs = append(errs, fmt.Errorf("%s: %w", tmplName, err))
			}
		}
		for _, b := range lc.Buffers {
			if err := applyBuffer(tmpl, b); err != nil {
				errs = append(errs, fmt.Errorf("%s: %w", tmplName, err))
			}
		}
	}
	return errors.Join(errs...)
}

func applyStage(tmpl *LineTemplateConfig, stage int, st ScenarioStageConfig) error {
	for _, f := range []struct {
		name string
		v    float64
	}{{"cycle_time_factor", st.CycleTimeFactor}, {"breakdown_factor", st.BreakdownFactor}, {"scrap_factor", st.ScrapFactor}} {
		if f.v < 0 {
			return fmt.Errorf("stage %d: %s %v must not be negative", stage, f.name, f.v)
		}
	}
	if st.AutoRestoreTimeMS < 0 {
		return fmt.Errorf("stage %d: auto_restore_time_ms must not be negative", stage)
	}
	idx := -1
	for i := range tmpl.Machines {
		if tmpl.Machines[i].Stage == stage {
			idx = i
		}
	}
	if idx < 0 {
		return fmt.Errorf("template has no stage %d", stage)
	}
	d := &tmpl.Machines[idx].Defaults
	if st.CycleTimeFactor > 0 {
		d.CycleTimeMS = scaleInt(d.CycleTimeMS, st.CycleTimeFactor)
	}
	if st.BreakdownFactor > 0 {
		d.BreakdownProbability = scaleRate(d.BreakdownProbability, st.BreakdownFactor)
	}
	if st.ScrapFactor > 0 {
		d.ScrapRate = scaleRate(d.ScrapRate, st.ScrapFactor)
	}
	if st.AutoRestoreTimeMS > 0 {
		d.AutoRestoreTimeMS = st.AutoRestoreTimeMS
	}
	for i := range tmpl.Recipes {
		ov, ok := tmpl.Recipes[i].Overrides[stage]
		if !ok {
			continue
		}
		if ov.CycleTimeMS != nil && st.CycleTimeFactor > 0 {
			v := scaleInt(*ov.CycleTimeMS, st.CycleTimeFactor)
			ov.CycleTimeMS = &v
		}
		if ov.BreakdownProbability != nil && st.BreakdownFactor > 0 {
			v := scaleRate(*ov.BreakdownProbability, st.BreakdownFactor)
			ov.BreakdownProbability = &v
		}
		if ov.ScrapRate != nil && st.ScrapFactor > 0 {
			v := scaleRate(*ov.ScrapRate, st.ScrapFactor)
			ov.ScrapRate = &v
		}
		tmpl.Recipes[i].Overrides[stage] = ov
	}
	return nil
}

func applyBuffer(tmpl *LineTemplateConfig, b BufferOverrideConfig) error {
	if b.Capacity < 1 {
		return fmt.Errorf("buffer %d-%d: capacity must be at least 1", b.FromStage, b.ToStage)
	}
	consecutive := false
	for i := 0; i+1 < len(tmpl.Machines); i++ {
		if tmpl.Machines[i].Stage == b.FromStage && tmpl.Machines[i+1].Stage == b.ToStage {
			consecutive = true
		}
	}
	if !consecutive {
		return fmt.Errorf("buffer %d-%d does not exist (stages must be neighbours)", b.FromStage, b.ToStage)
	}
	for i, ov := range tmpl.Buffers.Overrides {
		if ov.FromStage == b.FromStage && ov.ToStage == b.ToStage {
			tmpl.Buffers.Overrides[i].Capacity = b.Capacity
			return nil
		}
	}
	tmpl.Buffers.Overrides = append(tmpl.Buffers.Overrides, b)
	return nil
}

func scaleInt(v int, f float64) int { return int(math.Round(float64(v) * f)) }

// scaleRate scales a probability and keeps it a probability; rounding drops
// float noise such as 0.0040000000000000001.
func scaleRate(v, f float64) float64 {
	return math.Min(1, math.Round(v*f*1e9)/1e9)
}
