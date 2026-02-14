package types

import (
	"context"
	"time"

	"machine-simulator-2/internal/config"
	"machine-simulator-2/internal/machine"
)

func init() {
	machine.Register("profile_cutter", func(cfg config.MachineInstanceConfig) machine.Machine {
		return NewProfileCutter(cfg)
	})
}

// ProfileCutterMachine simulates a PVC/aluminum profile cutting machine for windows.
type ProfileCutterMachine struct {
	*machine.BaseMachine

	bladeSpeedRPM      float64
	cutLengthMM        float64
	feedRateMMS        float64
	bladeTempC         float64
	angleAccuracyDeg   float64
	cutDepthMM         float64
	clampPressureBar   float64
	materialHardnessHRC float64
	chipExtractionM3H  float64
	bladeWearPct       float64
	powerConsumptionKW float64
	vibrationMMS       float64
	coolantFlowLMin    float64
	surfaceRoughnessUM float64
	noiseLevelDB       float64
}

func NewProfileCutter(cfg config.MachineInstanceConfig) *ProfileCutterMachine {
	return &ProfileCutterMachine{
		BaseMachine:         machine.NewBaseMachine(cfg),
		bladeSpeedRPM:       3000.0,
		cutLengthMM:         1200.0,
		feedRateMMS:         50.0,
		bladeTempC:          45.0,
		angleAccuracyDeg:    0.05,
		cutDepthMM:          40.0,
		clampPressureBar:    5.0,
		materialHardnessHRC: 15.0,
		chipExtractionM3H:   120.0,
		bladeWearPct:        8.0,
		powerConsumptionKW:  5.5,
		vibrationMMS:        1.8,
		coolantFlowLMin:     3.0,
		surfaceRoughnessUM:  3.2,
		noiseLevelDB:        88.0,
	}
}

func (m *ProfileCutterMachine) Start(ctx context.Context) {
	go m.updateTags(ctx)
	m.BaseMachine.Start(ctx)
}

func (m *ProfileCutterMachine) updateTags(ctx context.Context) {
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
				m.bladeSpeedRPM += (rng.Float64()*2 - 1) * 10.0
				clampF(&m.bladeSpeedRPM, 2000.0, 4500.0)
				m.cutLengthMM += (rng.Float64()*2 - 1) * 0.1
				clampF(&m.cutLengthMM, 300.0, 3000.0)
				m.feedRateMMS += (rng.Float64()*2 - 1) * 0.5
				clampF(&m.feedRateMMS, 20.0, 100.0)
				m.bladeTempC += (rng.Float64()*2 - 1) * 0.3
				clampF(&m.bladeTempC, 25.0, 80.0)
				m.angleAccuracyDeg += (rng.Float64()*2 - 1) * 0.002
				clampF(&m.angleAccuracyDeg, 0.01, 0.2)
				m.cutDepthMM += (rng.Float64()*2 - 1) * 0.1
				clampF(&m.cutDepthMM, 10.0, 80.0)
				m.clampPressureBar += (rng.Float64()*2 - 1) * 0.1
				clampF(&m.clampPressureBar, 3.0, 8.0)
				m.materialHardnessHRC += (rng.Float64()*2 - 1) * 0.1
				clampF(&m.materialHardnessHRC, 8.0, 30.0)
				m.chipExtractionM3H += (rng.Float64()*2 - 1) * 2.0
				clampF(&m.chipExtractionM3H, 60.0, 200.0)
				m.bladeWearPct += rng.Float64() * 0.002
				clampF(&m.bladeWearPct, 0.0, 100.0)
				m.powerConsumptionKW += (rng.Float64()*2 - 1) * 0.1
				clampF(&m.powerConsumptionKW, 2.0, 10.0)
				m.vibrationMMS += (rng.Float64()*2 - 1) * 0.1
				clampF(&m.vibrationMMS, 0.5, 5.0)
				m.coolantFlowLMin += (rng.Float64()*2 - 1) * 0.05
				clampF(&m.coolantFlowLMin, 1.0, 8.0)
				m.surfaceRoughnessUM += (rng.Float64()*2 - 1) * 0.05
				clampF(&m.surfaceRoughnessUM, 0.8, 10.0)
				m.noiseLevelDB += (rng.Float64()*2 - 1) * 0.5
				clampF(&m.noiseLevelDB, 75.0, 100.0)
			}
			m.Unlock()
		}
	}
}

func (m *ProfileCutterMachine) GetTags() map[string]interface{} {
	m.RLock()
	defer m.RUnlock()
	tags := machine.BaseTagValues(m.BaseMachine)
	tags["blade_speed_rpm"] = m.bladeSpeedRPM
	tags["cut_length_mm"] = m.cutLengthMM
	tags["feed_rate_mm_s"] = m.feedRateMMS
	tags["blade_temp_c"] = m.bladeTempC
	tags["angle_accuracy_deg"] = m.angleAccuracyDeg
	tags["cut_depth_mm"] = m.cutDepthMM
	tags["clamp_pressure_bar"] = m.clampPressureBar
	tags["material_hardness_hrc"] = m.materialHardnessHRC
	tags["chip_extraction_m3_h"] = m.chipExtractionM3H
	tags["blade_wear_pct"] = m.bladeWearPct
	tags["power_consumption_kw"] = m.powerConsumptionKW
	tags["vibration_mm_s"] = m.vibrationMMS
	tags["coolant_flow_l_min"] = m.coolantFlowLMin
	tags["surface_roughness_um"] = m.surfaceRoughnessUM
	tags["noise_level_db"] = m.noiseLevelDB
	return tags
}

func (m *ProfileCutterMachine) GetTagDefinitions() []machine.TagDefinition {
	defs := machine.BaseTags()
	defs = append(defs,
		machine.TagDefinition{Name: "blade_speed_rpm", DisplayName: "Blade Speed (RPM)", NodeID: 7, DataType: "Double", Unit: "RPM", Description: "Blade rotation speed"},
		machine.TagDefinition{Name: "cut_length_mm", DisplayName: "Cut Length (mm)", NodeID: 8, DataType: "Double", Unit: "mm", Description: "Cut length"},
		machine.TagDefinition{Name: "feed_rate_mm_s", DisplayName: "Feed Rate (mm/s)", NodeID: 9, DataType: "Double", Unit: "mm/s", Description: "Feed rate"},
		machine.TagDefinition{Name: "blade_temp_c", DisplayName: "Blade Temp (°C)", NodeID: 10, DataType: "Double", Unit: "°C", Description: "Blade temperature"},
		machine.TagDefinition{Name: "angle_accuracy_deg", DisplayName: "Angle Accuracy (°)", NodeID: 11, DataType: "Double", Unit: "°", Description: "Cut angle accuracy"},
		machine.TagDefinition{Name: "cut_depth_mm", DisplayName: "Cut Depth (mm)", NodeID: 12, DataType: "Double", Unit: "mm", Description: "Cut depth"},
		machine.TagDefinition{Name: "clamp_pressure_bar", DisplayName: "Clamp Pressure (bar)", NodeID: 13, DataType: "Double", Unit: "bar", Description: "Clamp pressure"},
		machine.TagDefinition{Name: "material_hardness_hrc", DisplayName: "Material Hardness (HRC)", NodeID: 14, DataType: "Double", Unit: "HRC", Description: "Material hardness"},
		machine.TagDefinition{Name: "chip_extraction_m3_h", DisplayName: "Chip Extraction (m³/h)", NodeID: 15, DataType: "Double", Unit: "m³/h", Description: "Chip extraction airflow"},
		machine.TagDefinition{Name: "blade_wear_pct", DisplayName: "Blade Wear (%)", NodeID: 16, DataType: "Double", Unit: "%", Description: "Blade wear percentage"},
		machine.TagDefinition{Name: "power_consumption_kw", DisplayName: "Power Consumption (kW)", NodeID: 17, DataType: "Double", Unit: "kW", Description: "Power consumption"},
		machine.TagDefinition{Name: "vibration_mm_s", DisplayName: "Vibration (mm/s)", NodeID: 18, DataType: "Double", Unit: "mm/s", Description: "Vibration level"},
		machine.TagDefinition{Name: "coolant_flow_l_min", DisplayName: "Coolant Flow (L/min)", NodeID: 19, DataType: "Double", Unit: "L/min", Description: "Coolant flow rate"},
		machine.TagDefinition{Name: "surface_roughness_um", DisplayName: "Surface Roughness (µm)", NodeID: 20, DataType: "Double", Unit: "µm", Description: "Surface roughness"},
		machine.TagDefinition{Name: "noise_level_db", DisplayName: "Noise Level (dB)", NodeID: 21, DataType: "Double", Unit: "dB", Description: "Noise level"},
	)
	return defs
}
