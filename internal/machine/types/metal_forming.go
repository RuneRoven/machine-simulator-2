package types

import (
	"context"
	"time"

	"machine-simulator-2/internal/config"
	"machine-simulator-2/internal/machine"
)

func init() {
	machine.Register("metal_forming", func(cfg config.MachineInstanceConfig) machine.Machine {
		return NewMetalForming(cfg)
	})
}

// MetalFormingMachine simulates a hydraulic metal forming press.
type MetalFormingMachine struct {
	*machine.BaseMachine

	pressForceKN       float64
	hydraulicPressBar  float64
	dieTempC           float64
	strokePositionMM   float64
	toolWearPct        float64
	ramSpeedMMS        float64
	blankHolderForceKN float64
	materialThickMM    float64
	tonnagePct         float64
	lubricationPct     float64
	vibrationMMS       float64
	noiseLevelDB       float64
	motorCurrentA      float64
	hydraulicOilTempC  float64
	cycleEnergyKWH     float64
}

func NewMetalForming(cfg config.MachineInstanceConfig) *MetalFormingMachine {
	return &MetalFormingMachine{
		BaseMachine:        machine.NewBaseMachine(cfg),
		pressForceKN:       2500.0,
		hydraulicPressBar:  250.0,
		dieTempC:           45.0,
		strokePositionMM:   0.0,
		toolWearPct:        5.0,
		ramSpeedMMS:        120.0,
		blankHolderForceKN: 400.0,
		materialThickMM:    1.2,
		tonnagePct:         72.0,
		lubricationPct:     85.0,
		vibrationMMS:       2.5,
		noiseLevelDB:       92.0,
		motorCurrentA:      180.0,
		hydraulicOilTempC:  42.0,
		cycleEnergyKWH:     3.8,
	}
}

func (m *MetalFormingMachine) Start(ctx context.Context) {
	go m.updateTags(ctx)
	m.BaseMachine.Start(ctx)
}

func (m *MetalFormingMachine) updateTags(ctx context.Context) {
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
				m.pressForceKN += (rng.Float64()*2 - 1) * 10.0
				if m.pressForceKN < 2000 {
					m.pressForceKN = 2000
				} else if m.pressForceKN > 3000 {
					m.pressForceKN = 3000
				}
				m.hydraulicPressBar += (rng.Float64()*2 - 1) * 2.0
				if m.hydraulicPressBar < 200 {
					m.hydraulicPressBar = 200
				} else if m.hydraulicPressBar > 300 {
					m.hydraulicPressBar = 300
				}
				m.dieTempC += (rng.Float64()*2 - 1) * 0.5
				if m.dieTempC < 30 {
					m.dieTempC = 30
				} else if m.dieTempC > 80 {
					m.dieTempC = 80
				}
				m.strokePositionMM += (rng.Float64()*2 - 1) * 5.0
				if m.strokePositionMM < 0 {
					m.strokePositionMM = 0
				} else if m.strokePositionMM > 300 {
					m.strokePositionMM = 300
				}
				m.toolWearPct += rng.Float64() * 0.001
				if m.toolWearPct > 100 {
					m.toolWearPct = 100
				}
				m.ramSpeedMMS += (rng.Float64()*2 - 1) * 1.0
				if m.ramSpeedMMS < 80 {
					m.ramSpeedMMS = 80
				} else if m.ramSpeedMMS > 160 {
					m.ramSpeedMMS = 160
				}
				m.blankHolderForceKN += (rng.Float64()*2 - 1) * 5.0
				if m.blankHolderForceKN < 300 {
					m.blankHolderForceKN = 300
				} else if m.blankHolderForceKN > 500 {
					m.blankHolderForceKN = 500
				}
				m.materialThickMM += (rng.Float64()*2 - 1) * 0.01
				if m.materialThickMM < 0.8 {
					m.materialThickMM = 0.8
				} else if m.materialThickMM > 2.0 {
					m.materialThickMM = 2.0
				}
				m.tonnagePct += (rng.Float64()*2 - 1) * 1.0
				if m.tonnagePct < 50 {
					m.tonnagePct = 50
				} else if m.tonnagePct > 95 {
					m.tonnagePct = 95
				}
				m.lubricationPct -= rng.Float64() * 0.01
				if m.lubricationPct < 20 {
					m.lubricationPct = 20
				}
				m.vibrationMMS += (rng.Float64()*2 - 1) * 0.2
				if m.vibrationMMS < 0.5 {
					m.vibrationMMS = 0.5
				} else if m.vibrationMMS > 10.0 {
					m.vibrationMMS = 10.0
				}
				m.noiseLevelDB += (rng.Float64()*2 - 1) * 0.5
				if m.noiseLevelDB < 80 {
					m.noiseLevelDB = 80
				} else if m.noiseLevelDB > 105 {
					m.noiseLevelDB = 105
				}
				m.motorCurrentA += (rng.Float64()*2 - 1) * 2.0
				if m.motorCurrentA < 120 {
					m.motorCurrentA = 120
				} else if m.motorCurrentA > 250 {
					m.motorCurrentA = 250
				}
				m.hydraulicOilTempC += (rng.Float64()*2 - 1) * 0.3
				if m.hydraulicOilTempC < 30 {
					m.hydraulicOilTempC = 30
				} else if m.hydraulicOilTempC > 65 {
					m.hydraulicOilTempC = 65
				}
				m.cycleEnergyKWH += (rng.Float64()*2 - 1) * 0.1
				if m.cycleEnergyKWH < 2.0 {
					m.cycleEnergyKWH = 2.0
				} else if m.cycleEnergyKWH > 6.0 {
					m.cycleEnergyKWH = 6.0
				}
			}
			m.Unlock()
		}
	}
}

func (m *MetalFormingMachine) GetTags() map[string]interface{} {
	m.RLock()
	defer m.RUnlock()
	tags := machine.BaseTagValues(m.BaseMachine)
	tags["press_force_kn"] = m.pressForceKN
	tags["hydraulic_pressure_bar"] = m.hydraulicPressBar
	tags["die_temperature_c"] = m.dieTempC
	tags["stroke_position_mm"] = m.strokePositionMM
	tags["tool_wear_pct"] = m.toolWearPct
	tags["ram_speed_mm_s"] = m.ramSpeedMMS
	tags["blank_holder_force_kn"] = m.blankHolderForceKN
	tags["material_thickness_mm"] = m.materialThickMM
	tags["tonnage_pct"] = m.tonnagePct
	tags["lubrication_level_pct"] = m.lubricationPct
	tags["vibration_mm_s"] = m.vibrationMMS
	tags["noise_level_db"] = m.noiseLevelDB
	tags["motor_current_a"] = m.motorCurrentA
	tags["hydraulic_oil_temp_c"] = m.hydraulicOilTempC
	tags["cycle_energy_kwh"] = m.cycleEnergyKWH
	return tags
}

func (m *MetalFormingMachine) GetTagDefinitions() []machine.TagDefinition {
	defs := machine.BaseTags()
	defs = append(defs,
		machine.TagDefinition{Name: "press_force_kn", DisplayName: "Press Force (kN)", NodeID: 7, DataType: "Double", Unit: "kN", Description: "Press force"},
		machine.TagDefinition{Name: "hydraulic_pressure_bar", DisplayName: "Hydraulic Pressure (bar)", NodeID: 8, DataType: "Double", Unit: "bar", Description: "Hydraulic system pressure"},
		machine.TagDefinition{Name: "die_temperature_c", DisplayName: "Die Temperature (\u00b0C)", NodeID: 9, DataType: "Double", Unit: "°C", Description: "Die temperature"},
		machine.TagDefinition{Name: "stroke_position_mm", DisplayName: "Stroke Position (mm)", NodeID: 10, DataType: "Double", Unit: "mm", Description: "Ram stroke position"},
		machine.TagDefinition{Name: "tool_wear_pct", DisplayName: "Tool Wear (%)", NodeID: 11, DataType: "Double", Unit: "%", Description: "Tool wear percentage"},
		machine.TagDefinition{Name: "ram_speed_mm_s", DisplayName: "Ram Speed (mm/s)", NodeID: 12, DataType: "Double", Unit: "mm/s", Description: "Ram speed"},
		machine.TagDefinition{Name: "blank_holder_force_kn", DisplayName: "Blank Holder Force (kN)", NodeID: 13, DataType: "Double", Unit: "kN", Description: "Blank holder force"},
		machine.TagDefinition{Name: "material_thickness_mm", DisplayName: "Material Thickness (mm)", NodeID: 14, DataType: "Double", Unit: "mm", Description: "Material thickness"},
		machine.TagDefinition{Name: "tonnage_pct", DisplayName: "Tonnage Utilization (%)", NodeID: 15, DataType: "Double", Unit: "%", Description: "Tonnage utilization"},
		machine.TagDefinition{Name: "lubrication_level_pct", DisplayName: "Lubrication Level (%)", NodeID: 16, DataType: "Double", Unit: "%", Description: "Lubrication level"},
		machine.TagDefinition{Name: "vibration_mm_s", DisplayName: "Vibration Level (mm/s)", NodeID: 17, DataType: "Double", Unit: "mm/s", Description: "Vibration level"},
		machine.TagDefinition{Name: "noise_level_db", DisplayName: "Noise Level (dB)", NodeID: 18, DataType: "Double", Unit: "dB", Description: "Noise level"},
		machine.TagDefinition{Name: "motor_current_a", DisplayName: "Motor Current (A)", NodeID: 19, DataType: "Double", Unit: "A", Description: "Motor current draw"},
		machine.TagDefinition{Name: "hydraulic_oil_temp_c", DisplayName: "Hydraulic Oil Temp (\u00b0C)", NodeID: 20, DataType: "Double", Unit: "°C", Description: "Hydraulic oil temperature"},
		machine.TagDefinition{Name: "cycle_energy_kwh", DisplayName: "Cycle Energy (kWh)", NodeID: 21, DataType: "Double", Unit: "kWh", Description: "Energy consumed per cycle"},
	)
	return defs
}
