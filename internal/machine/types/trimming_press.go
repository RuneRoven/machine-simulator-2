package types

import (
	"context"
	"time"

	"machine-simulator-2/internal/config"
	"machine-simulator-2/internal/machine"
)

func init() {
	machine.Register("trimming_press", func(cfg config.MachineInstanceConfig) machine.Machine {
		return NewTrimmingPress(cfg)
	})
}

// TrimmingPressMachine simulates a plastic part trimming press.
type TrimmingPressMachine struct {
	*machine.BaseMachine

	pressForceKN      float64
	trimAccuracyMM    float64
	feedRateMMS       float64
	dieClearanceMM    float64
	strokeSpeedSPM    float64
	materialThickMM   float64
	dieTempC          float64
	stripperForceN    float64
	slugEjectionPct   float64
	lubricationPct    float64
	dieWearPct        float64
	punchAlignmentUM  float64
	pressEnergyJ      float64
	vibrationMMS      float64
	noiseLevelDB      float64
}

func NewTrimmingPress(cfg config.MachineInstanceConfig) *TrimmingPressMachine {
	return &TrimmingPressMachine{
		BaseMachine:      machine.NewBaseMachine(cfg),
		pressForceKN:     50.0,
		trimAccuracyMM:   0.05,
		feedRateMMS:      40.0,
		dieClearanceMM:   0.08,
		strokeSpeedSPM:   60.0,
		materialThickMM:  2.5,
		dieTempC:         35.0,
		stripperForceN:   500.0,
		slugEjectionPct:  98.0,
		lubricationPct:   80.0,
		dieWearPct:       5.0,
		punchAlignmentUM: 15.0,
		pressEnergyJ:     120.0,
		vibrationMMS:     2.0,
		noiseLevelDB:     90.0,
	}
}

func (m *TrimmingPressMachine) Start(ctx context.Context) {
	go m.updateTags(ctx)
	m.BaseMachine.Start(ctx)
}

func (m *TrimmingPressMachine) updateTags(ctx context.Context) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.RLock()
			state := m.GetStateUnlocked()
			m.RUnlock()

			m.Lock()
			if state == machine.StateRunning {
				rng := m.Rng()
				m.pressForceKN += (rng.Float64()*2 - 1) * 0.5
				clampF(&m.pressForceKN, 10.0, 150.0)
				m.trimAccuracyMM += (rng.Float64()*2 - 1) * 0.002
				clampF(&m.trimAccuracyMM, 0.01, 0.2)
				m.feedRateMMS += (rng.Float64()*2 - 1) * 0.5
				clampF(&m.feedRateMMS, 15.0, 80.0)
				m.dieClearanceMM += (rng.Float64()*2 - 1) * 0.002
				clampF(&m.dieClearanceMM, 0.02, 0.3)
				m.strokeSpeedSPM += (rng.Float64()*2 - 1) * 0.5
				clampF(&m.strokeSpeedSPM, 20.0, 120.0)
				m.materialThickMM += (rng.Float64()*2 - 1) * 0.02
				clampF(&m.materialThickMM, 0.5, 8.0)
				m.dieTempC += (rng.Float64()*2 - 1) * 0.2
				clampF(&m.dieTempC, 20.0, 60.0)
				m.stripperForceN += (rng.Float64()*2 - 1) * 5.0
				clampF(&m.stripperForceN, 100.0, 1500.0)
				m.slugEjectionPct += (rng.Float64()*2 - 1) * 0.1
				clampF(&m.slugEjectionPct, 85.0, 100.0)
				m.lubricationPct -= rng.Float64() * 0.01
				clampF(&m.lubricationPct, 20.0, 100.0)
				m.dieWearPct += rng.Float64() * 0.001
				clampF(&m.dieWearPct, 0.0, 100.0)
				m.punchAlignmentUM += (rng.Float64()*2 - 1) * 0.3
				clampF(&m.punchAlignmentUM, 3.0, 50.0)
				m.pressEnergyJ += (rng.Float64()*2 - 1) * 2.0
				clampF(&m.pressEnergyJ, 30.0, 300.0)
				m.vibrationMMS += (rng.Float64()*2 - 1) * 0.1
				clampF(&m.vibrationMMS, 0.5, 8.0)
				m.noiseLevelDB += (rng.Float64()*2 - 1) * 0.3
				clampF(&m.noiseLevelDB, 78.0, 105.0)
			}
			m.Unlock()
		}
	}
}

func (m *TrimmingPressMachine) GetTags() map[string]interface{} {
	m.RLock()
	defer m.RUnlock()
	tags := machine.BaseTagValues(m.BaseMachine)
	tags["press_force_kn"] = m.pressForceKN
	tags["trim_accuracy_mm"] = m.trimAccuracyMM
	tags["feed_rate_mm_s"] = m.feedRateMMS
	tags["die_clearance_mm"] = m.dieClearanceMM
	tags["stroke_speed_spm"] = m.strokeSpeedSPM
	tags["material_thickness_mm"] = m.materialThickMM
	tags["die_temp_c"] = m.dieTempC
	tags["stripper_force_n"] = m.stripperForceN
	tags["slug_ejection_pct"] = m.slugEjectionPct
	tags["lubrication_level_pct"] = m.lubricationPct
	tags["die_wear_pct"] = m.dieWearPct
	tags["punch_alignment_um"] = m.punchAlignmentUM
	tags["press_energy_j"] = m.pressEnergyJ
	tags["vibration_mm_s"] = m.vibrationMMS
	tags["noise_level_db"] = m.noiseLevelDB
	return tags
}

func (m *TrimmingPressMachine) GetTagDefinitions() []machine.TagDefinition {
	defs := machine.BaseTags()
	defs = append(defs,
		machine.TagDefinition{Name: "press_force_kn", DisplayName: "Press Force (kN)", NodeID: 7, DataType: "Double", Unit: "kN", Description: "Press force"},
		machine.TagDefinition{Name: "trim_accuracy_mm", DisplayName: "Trim Accuracy (mm)", NodeID: 8, DataType: "Double", Unit: "mm", Description: "Trim accuracy"},
		machine.TagDefinition{Name: "feed_rate_mm_s", DisplayName: "Feed Rate (mm/s)", NodeID: 9, DataType: "Double", Unit: "mm/s", Description: "Feed rate"},
		machine.TagDefinition{Name: "die_clearance_mm", DisplayName: "Die Clearance (mm)", NodeID: 10, DataType: "Double", Unit: "mm", Description: "Die clearance"},
		machine.TagDefinition{Name: "stroke_speed_spm", DisplayName: "Stroke Speed (SPM)", NodeID: 11, DataType: "Double", Unit: "SPM", Description: "Stroke speed"},
		machine.TagDefinition{Name: "material_thickness_mm", DisplayName: "Material Thickness (mm)", NodeID: 12, DataType: "Double", Unit: "mm", Description: "Material thickness"},
		machine.TagDefinition{Name: "die_temp_c", DisplayName: "Die Temperature (\u00b0C)", NodeID: 13, DataType: "Double", Unit: "°C", Description: "Die temperature"},
		machine.TagDefinition{Name: "stripper_force_n", DisplayName: "Stripper Force (N)", NodeID: 14, DataType: "Double", Unit: "N", Description: "Stripper force"},
		machine.TagDefinition{Name: "slug_ejection_pct", DisplayName: "Slug Ejection (%)", NodeID: 15, DataType: "Double", Unit: "%", Description: "Slug ejection rate"},
		machine.TagDefinition{Name: "lubrication_level_pct", DisplayName: "Lubrication Level (%)", NodeID: 16, DataType: "Double", Unit: "%", Description: "Lubrication level"},
		machine.TagDefinition{Name: "die_wear_pct", DisplayName: "Die Wear (%)", NodeID: 17, DataType: "Double", Unit: "%", Description: "Die wear percentage"},
		machine.TagDefinition{Name: "punch_alignment_um", DisplayName: "Punch Alignment (\u00b5m)", NodeID: 18, DataType: "Double", Unit: "µm", Description: "Punch alignment"},
		machine.TagDefinition{Name: "press_energy_j", DisplayName: "Press Energy (J)", NodeID: 19, DataType: "Double", Unit: "J", Description: "Press energy per stroke"},
		machine.TagDefinition{Name: "vibration_mm_s", DisplayName: "Vibration (mm/s)", NodeID: 20, DataType: "Double", Unit: "mm/s", Description: "Vibration level"},
		machine.TagDefinition{Name: "noise_level_db", DisplayName: "Noise Level (dB)", NodeID: 21, DataType: "Double", Unit: "dB", Description: "Noise level"},
	)
	return defs
}
