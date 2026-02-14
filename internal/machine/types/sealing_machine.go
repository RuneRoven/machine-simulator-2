package types

import (
	"context"
	"time"

	"machine-simulator-2/internal/config"
	"machine-simulator-2/internal/machine"
)

func init() {
	machine.Register("sealing_machine", func(cfg config.MachineInstanceConfig) machine.Machine {
		return NewSealingMachine(cfg)
	})
}

// SealingMachineMachine simulates a heat sealing machine for packaging.
type SealingMachineMachine struct {
	*machine.BaseMachine

	sealTempC         float64
	sealPressureBar   float64
	sealTimeMS        float64
	conveyorSpeedMMin float64
	jawGapMM          float64
	sealStrengthN     float64
	peelForceN        float64
	sealWidthMM       float64
	coolingWaterTempC float64
	filmTensionN      float64
	rejectCount       float64
	vacuumLevelMbar   float64
	gasFlushFlowLMin  float64
	leakRateMbarLS    float64
	heaterPowerPct    float64
}

func NewSealingMachine(cfg config.MachineInstanceConfig) *SealingMachineMachine {
	return &SealingMachineMachine{
		BaseMachine:       machine.NewBaseMachine(cfg),
		sealTempC:         180.0,
		sealPressureBar:   3.5,
		sealTimeMS:        500.0,
		conveyorSpeedMMin: 12.0,
		jawGapMM:          0.5,
		sealStrengthN:     25.0,
		peelForceN:        8.0,
		sealWidthMM:       10.0,
		coolingWaterTempC: 18.0,
		filmTensionN:      15.0,
		rejectCount:       0.0,
		vacuumLevelMbar:   5.0,
		gasFlushFlowLMin:  2.0,
		leakRateMbarLS:    0.01,
		heaterPowerPct:    75.0,
	}
}

func (m *SealingMachineMachine) Start(ctx context.Context) {
	go m.updateTags(ctx)
	m.BaseMachine.Start(ctx)
}

func (m *SealingMachineMachine) updateTags(ctx context.Context) {
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
				m.sealTempC += (rng.Float64()*2 - 1) * 0.5
				clampF(&m.sealTempC, 140.0, 220.0)
				m.sealPressureBar += (rng.Float64()*2 - 1) * 0.1
				clampF(&m.sealPressureBar, 2.0, 6.0)
				m.sealTimeMS += (rng.Float64()*2 - 1) * 5.0
				clampF(&m.sealTimeMS, 200.0, 1000.0)
				m.conveyorSpeedMMin += (rng.Float64()*2 - 1) * 0.2
				clampF(&m.conveyorSpeedMMin, 5.0, 25.0)
				m.jawGapMM += (rng.Float64()*2 - 1) * 0.02
				clampF(&m.jawGapMM, 0.1, 2.0)
				m.sealStrengthN += (rng.Float64()*2 - 1) * 0.5
				clampF(&m.sealStrengthN, 10.0, 50.0)
				m.peelForceN += (rng.Float64()*2 - 1) * 0.2
				clampF(&m.peelForceN, 3.0, 20.0)
				m.sealWidthMM += (rng.Float64()*2 - 1) * 0.05
				clampF(&m.sealWidthMM, 5.0, 20.0)
				m.coolingWaterTempC += (rng.Float64()*2 - 1) * 0.1
				clampF(&m.coolingWaterTempC, 10.0, 25.0)
				m.filmTensionN += (rng.Float64()*2 - 1) * 0.3
				clampF(&m.filmTensionN, 5.0, 30.0)
				if rng.Float64() < 0.005 {
					m.rejectCount++
				}
				m.vacuumLevelMbar += (rng.Float64()*2 - 1) * 0.2
				clampF(&m.vacuumLevelMbar, 1.0, 20.0)
				m.gasFlushFlowLMin += (rng.Float64()*2 - 1) * 0.05
				clampF(&m.gasFlushFlowLMin, 0.5, 5.0)
				m.leakRateMbarLS += (rng.Float64()*2 - 1) * 0.001
				clampF(&m.leakRateMbarLS, 0.001, 0.1)
				m.heaterPowerPct += (rng.Float64()*2 - 1) * 0.5
				clampF(&m.heaterPowerPct, 40.0, 100.0)
			}
			m.Unlock()
		}
	}
}

func (m *SealingMachineMachine) GetTags() map[string]interface{} {
	m.RLock()
	defer m.RUnlock()
	tags := machine.BaseTagValues(m.BaseMachine)
	tags["seal_temp_c"] = m.sealTempC
	tags["seal_pressure_bar"] = m.sealPressureBar
	tags["seal_time_ms"] = m.sealTimeMS
	tags["conveyor_speed_m_min"] = m.conveyorSpeedMMin
	tags["jaw_gap_mm"] = m.jawGapMM
	tags["seal_strength_n"] = m.sealStrengthN
	tags["peel_force_n"] = m.peelForceN
	tags["seal_width_mm"] = m.sealWidthMM
	tags["cooling_water_temp_c"] = m.coolingWaterTempC
	tags["film_tension_n"] = m.filmTensionN
	tags["reject_count"] = m.rejectCount
	tags["vacuum_level_mbar"] = m.vacuumLevelMbar
	tags["gas_flush_flow_l_min"] = m.gasFlushFlowLMin
	tags["leak_rate_mbar_l_s"] = m.leakRateMbarLS
	tags["heater_power_pct"] = m.heaterPowerPct
	return tags
}

func (m *SealingMachineMachine) GetTagDefinitions() []machine.TagDefinition {
	defs := machine.BaseTags()
	defs = append(defs,
		machine.TagDefinition{Name: "seal_temp_c", DisplayName: "Seal Temp (°C)", NodeID: 7, DataType: "Double", Unit: "°C", Description: "Seal bar temperature"},
		machine.TagDefinition{Name: "seal_pressure_bar", DisplayName: "Seal Pressure (bar)", NodeID: 8, DataType: "Double", Unit: "bar", Description: "Seal pressure"},
		machine.TagDefinition{Name: "seal_time_ms", DisplayName: "Seal Time (ms)", NodeID: 9, DataType: "Double", Unit: "ms", Description: "Seal dwell time"},
		machine.TagDefinition{Name: "conveyor_speed_m_min", DisplayName: "Conveyor Speed (m/min)", NodeID: 10, DataType: "Double", Unit: "m/min", Description: "Conveyor speed"},
		machine.TagDefinition{Name: "jaw_gap_mm", DisplayName: "Jaw Gap (mm)", NodeID: 11, DataType: "Double", Unit: "mm", Description: "Jaw gap"},
		machine.TagDefinition{Name: "seal_strength_n", DisplayName: "Seal Strength (N)", NodeID: 12, DataType: "Double", Unit: "N", Description: "Seal strength"},
		machine.TagDefinition{Name: "peel_force_n", DisplayName: "Peel Force (N)", NodeID: 13, DataType: "Double", Unit: "N", Description: "Peel force"},
		machine.TagDefinition{Name: "seal_width_mm", DisplayName: "Seal Width (mm)", NodeID: 14, DataType: "Double", Unit: "mm", Description: "Seal width"},
		machine.TagDefinition{Name: "cooling_water_temp_c", DisplayName: "Cooling Water Temp (°C)", NodeID: 15, DataType: "Double", Unit: "°C", Description: "Cooling water temperature"},
		machine.TagDefinition{Name: "film_tension_n", DisplayName: "Film Tension (N)", NodeID: 16, DataType: "Double", Unit: "N", Description: "Film tension"},
		machine.TagDefinition{Name: "reject_count", DisplayName: "Reject Count", NodeID: 17, DataType: "Double", Unit: "", Description: "Rejected seals"},
		machine.TagDefinition{Name: "vacuum_level_mbar", DisplayName: "Vacuum Level (mbar)", NodeID: 18, DataType: "Double", Unit: "mbar", Description: "Vacuum level for MAP"},
		machine.TagDefinition{Name: "gas_flush_flow_l_min", DisplayName: "Gas Flush Flow (L/min)", NodeID: 19, DataType: "Double", Unit: "L/min", Description: "Gas flush flow rate"},
		machine.TagDefinition{Name: "leak_rate_mbar_l_s", DisplayName: "Leak Rate (mbar·L/s)", NodeID: 20, DataType: "Double", Unit: "mbar·L/s", Description: "Leak rate"},
		machine.TagDefinition{Name: "heater_power_pct", DisplayName: "Heater Power (%)", NodeID: 21, DataType: "Double", Unit: "%", Description: "Heater power percentage"},
	)
	return defs
}
