package types

import (
	"context"
	"time"

	"machine-simulator-2/internal/config"
	"machine-simulator-2/internal/machine"
)

func init() {
	machine.Register("reflow_oven", func(cfg config.MachineInstanceConfig) machine.Machine {
		return NewReflowOven(cfg)
	})
}

// ReflowOvenMachine simulates a multi-zone reflow soldering oven.
type ReflowOvenMachine struct {
	*machine.BaseMachine

	zone1TempC          float64 // preheat
	zone2TempC          float64 // soak
	zone3TempC          float64 // reflow
	zone4TempC          float64 // cooling
	conveyorSpeedCMMin  float64
	oxygenLevelPPM      float64
	nitrogenFlowLMin    float64
	peakTempC           float64
	timeAboveLiquidusS  float64
	heatingRateCS       float64
	coolingRateCS       float64
	boardCount          float64
	powerConsumptionKW  float64
	exhaustTempC        float64
	beltWidthMM         float64
}

func NewReflowOven(cfg config.MachineInstanceConfig) *ReflowOvenMachine {
	return &ReflowOvenMachine{
		BaseMachine:        machine.NewBaseMachine(cfg),
		zone1TempC:         150.0,
		zone2TempC:         180.0,
		zone3TempC:         245.0,
		zone4TempC:         100.0,
		conveyorSpeedCMMin: 80.0,
		oxygenLevelPPM:     500.0,
		nitrogenFlowLMin:   40.0,
		peakTempC:          250.0,
		timeAboveLiquidusS: 60.0,
		heatingRateCS:      2.0,
		coolingRateCS:      3.5,
		boardCount:         0.0,
		powerConsumptionKW: 18.0,
		exhaustTempC:       85.0,
		beltWidthMM:        450.0,
	}
}

func (m *ReflowOvenMachine) Start(ctx context.Context) {
	go m.updateTags(ctx)
	m.BaseMachine.Start(ctx)
}

func (m *ReflowOvenMachine) updateTags(ctx context.Context) {
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
				m.zone1TempC += (rng.Float64()*2 - 1) * 0.5
				clampF(&m.zone1TempC, 130.0, 170.0)
				m.zone2TempC += (rng.Float64()*2 - 1) * 0.5
				clampF(&m.zone2TempC, 160.0, 200.0)
				m.zone3TempC += (rng.Float64()*2 - 1) * 0.5
				clampF(&m.zone3TempC, 230.0, 260.0)
				m.zone4TempC += (rng.Float64()*2 - 1) * 0.5
				clampF(&m.zone4TempC, 50.0, 150.0)
				m.conveyorSpeedCMMin += (rng.Float64()*2 - 1) * 0.5
				clampF(&m.conveyorSpeedCMMin, 40.0, 120.0)
				m.oxygenLevelPPM += (rng.Float64()*2 - 1) * 10.0
				clampF(&m.oxygenLevelPPM, 50.0, 2000.0)
				m.nitrogenFlowLMin += (rng.Float64()*2 - 1) * 0.5
				clampF(&m.nitrogenFlowLMin, 20.0, 80.0)
				m.peakTempC += (rng.Float64()*2 - 1) * 0.5
				clampF(&m.peakTempC, 240.0, 265.0)
				m.timeAboveLiquidusS += (rng.Float64()*2 - 1) * 1.0
				clampF(&m.timeAboveLiquidusS, 30.0, 120.0)
				m.heatingRateCS += (rng.Float64()*2 - 1) * 0.05
				clampF(&m.heatingRateCS, 1.0, 4.0)
				m.coolingRateCS += (rng.Float64()*2 - 1) * 0.1
				clampF(&m.coolingRateCS, 1.0, 6.0)
				m.boardCount += rng.Float64() * 0.5
				m.powerConsumptionKW += (rng.Float64()*2 - 1) * 0.3
				clampF(&m.powerConsumptionKW, 10.0, 30.0)
				m.exhaustTempC += (rng.Float64()*2 - 1) * 0.5
				clampF(&m.exhaustTempC, 50.0, 120.0)
				m.beltWidthMM += (rng.Float64()*2 - 1) * 0.1
				clampF(&m.beltWidthMM, 300.0, 600.0)
			}
			m.Unlock()
		}
	}
}

func (m *ReflowOvenMachine) GetTags() map[string]interface{} {
	m.RLock()
	defer m.RUnlock()
	tags := machine.BaseTagValues(m.BaseMachine)
	tags["zone1_temp_c"] = m.zone1TempC
	tags["zone2_temp_c"] = m.zone2TempC
	tags["zone3_temp_c"] = m.zone3TempC
	tags["zone4_temp_c"] = m.zone4TempC
	tags["conveyor_speed_cm_min"] = m.conveyorSpeedCMMin
	tags["oxygen_level_ppm"] = m.oxygenLevelPPM
	tags["nitrogen_flow_l_min"] = m.nitrogenFlowLMin
	tags["peak_temp_c"] = m.peakTempC
	tags["time_above_liquidus_s"] = m.timeAboveLiquidusS
	tags["heating_rate_c_s"] = m.heatingRateCS
	tags["cooling_rate_c_s"] = m.coolingRateCS
	tags["board_count"] = m.boardCount
	tags["power_consumption_kw"] = m.powerConsumptionKW
	tags["exhaust_temp_c"] = m.exhaustTempC
	tags["belt_width_mm"] = m.beltWidthMM
	return tags
}

func (m *ReflowOvenMachine) GetTagDefinitions() []machine.TagDefinition {
	defs := machine.BaseTags()
	defs = append(defs,
		machine.TagDefinition{Name: "zone1_temp_c", DisplayName: "Preheat Zone Temp (\u00b0C)", NodeID: 7, DataType: "Double", Unit: "°C", Description: "Preheat zone temperature"},
		machine.TagDefinition{Name: "zone2_temp_c", DisplayName: "Soak Zone Temp (\u00b0C)", NodeID: 8, DataType: "Double", Unit: "°C", Description: "Soak zone temperature"},
		machine.TagDefinition{Name: "zone3_temp_c", DisplayName: "Reflow Zone Temp (\u00b0C)", NodeID: 9, DataType: "Double", Unit: "°C", Description: "Reflow zone temperature"},
		machine.TagDefinition{Name: "zone4_temp_c", DisplayName: "Cooling Zone Temp (\u00b0C)", NodeID: 10, DataType: "Double", Unit: "°C", Description: "Cooling zone temperature"},
		machine.TagDefinition{Name: "conveyor_speed_cm_min", DisplayName: "Conveyor Speed (cm/min)", NodeID: 11, DataType: "Double", Unit: "cm/min", Description: "Conveyor speed"},
		machine.TagDefinition{Name: "oxygen_level_ppm", DisplayName: "Oxygen Level (ppm)", NodeID: 12, DataType: "Double", Unit: "ppm", Description: "Oxygen level in oven"},
		machine.TagDefinition{Name: "nitrogen_flow_l_min", DisplayName: "Nitrogen Flow (L/min)", NodeID: 13, DataType: "Double", Unit: "L/min", Description: "Nitrogen flow rate"},
		machine.TagDefinition{Name: "peak_temp_c", DisplayName: "Peak Reflow Temp (\u00b0C)", NodeID: 14, DataType: "Double", Unit: "°C", Description: "Peak reflow temperature"},
		machine.TagDefinition{Name: "time_above_liquidus_s", DisplayName: "Time Above Liquidus (s)", NodeID: 15, DataType: "Double", Unit: "s", Description: "Time above solder liquidus"},
		machine.TagDefinition{Name: "heating_rate_c_s", DisplayName: "Heating Rate (\u00b0C/s)", NodeID: 16, DataType: "Double", Unit: "°C/s", Description: "Heating rate"},
		machine.TagDefinition{Name: "cooling_rate_c_s", DisplayName: "Cooling Rate (\u00b0C/s)", NodeID: 17, DataType: "Double", Unit: "°C/s", Description: "Cooling rate"},
		machine.TagDefinition{Name: "board_count", DisplayName: "Board Count", NodeID: 18, DataType: "Double", Unit: "", Description: "Boards processed"},
		machine.TagDefinition{Name: "power_consumption_kw", DisplayName: "Power Consumption (kW)", NodeID: 19, DataType: "Double", Unit: "kW", Description: "Power consumption"},
		machine.TagDefinition{Name: "exhaust_temp_c", DisplayName: "Exhaust Temp (\u00b0C)", NodeID: 20, DataType: "Double", Unit: "°C", Description: "Exhaust temperature"},
		machine.TagDefinition{Name: "belt_width_mm", DisplayName: "Belt Width (mm)", NodeID: 21, DataType: "Double", Unit: "mm", Description: "Belt width"},
	)
	return defs
}
