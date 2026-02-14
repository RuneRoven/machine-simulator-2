package types

import (
	"context"
	"time"

	"machine-simulator-2/internal/config"
	"machine-simulator-2/internal/machine"
)

func init() {
	machine.Register("cnc_router", func(cfg config.MachineInstanceConfig) machine.Machine {
		return NewCNCRouter(cfg)
	})
}

// CNCRouterMachine simulates a CNC router for furniture manufacturing.
type CNCRouterMachine struct {
	*machine.BaseMachine

	spindleSpeedRPM     float64
	xPositionMM         float64
	yPositionMM         float64
	zPositionMM         float64
	cutDepthMM          float64
	feedRateMMMin       float64
	dustExtractionM3H   float64
	toolWearPct         float64
	powerConsumptionKW  float64
	vibrationMMS        float64
	vacuumHoldDownKPa   float64
	surfaceFinishUM     float64
	toolTempC           float64
	coolantFlowLMin     float64
	materialRemovalCM3M float64
}

func NewCNCRouter(cfg config.MachineInstanceConfig) *CNCRouterMachine {
	return &CNCRouterMachine{
		BaseMachine:         machine.NewBaseMachine(cfg),
		spindleSpeedRPM:     18000.0,
		xPositionMM:         500.0,
		yPositionMM:         300.0,
		zPositionMM:         -5.0,
		cutDepthMM:          6.0,
		feedRateMMMin:       8000.0,
		dustExtractionM3H:   2000.0,
		toolWearPct:         10.0,
		powerConsumptionKW:  7.5,
		vibrationMMS:        1.2,
		vacuumHoldDownKPa:   -70.0,
		surfaceFinishUM:     5.0,
		toolTempC:           55.0,
		coolantFlowLMin:     0.0,
		materialRemovalCM3M: 150.0,
	}
}

func (m *CNCRouterMachine) Start(ctx context.Context) {
	go m.updateTags(ctx)
	m.BaseMachine.Start(ctx)
}

func (m *CNCRouterMachine) updateTags(ctx context.Context) {
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
				m.spindleSpeedRPM += (rng.Float64()*2 - 1) * 50.0
				clampF(&m.spindleSpeedRPM, 6000.0, 24000.0)
				m.xPositionMM += (rng.Float64()*2 - 1) * 20.0
				clampF(&m.xPositionMM, 0.0, 2500.0)
				m.yPositionMM += (rng.Float64()*2 - 1) * 15.0
				clampF(&m.yPositionMM, 0.0, 1250.0)
				m.zPositionMM += (rng.Float64()*2 - 1) * 1.0
				clampF(&m.zPositionMM, -50.0, 10.0)
				m.cutDepthMM += (rng.Float64()*2 - 1) * 0.1
				clampF(&m.cutDepthMM, 1.0, 25.0)
				m.feedRateMMMin += (rng.Float64()*2 - 1) * 50.0
				clampF(&m.feedRateMMMin, 2000.0, 15000.0)
				m.dustExtractionM3H += (rng.Float64()*2 - 1) * 20.0
				clampF(&m.dustExtractionM3H, 1000.0, 4000.0)
				m.toolWearPct += rng.Float64() * 0.002
				clampF(&m.toolWearPct, 0.0, 100.0)
				m.powerConsumptionKW += (rng.Float64()*2 - 1) * 0.2
				clampF(&m.powerConsumptionKW, 3.0, 15.0)
				m.vibrationMMS += (rng.Float64()*2 - 1) * 0.1
				clampF(&m.vibrationMMS, 0.3, 5.0)
				m.vacuumHoldDownKPa += (rng.Float64()*2 - 1) * 0.5
				clampF(&m.vacuumHoldDownKPa, -90.0, -40.0)
				m.surfaceFinishUM += (rng.Float64()*2 - 1) * 0.2
				clampF(&m.surfaceFinishUM, 1.0, 15.0)
				m.toolTempC += (rng.Float64()*2 - 1) * 0.5
				clampF(&m.toolTempC, 25.0, 90.0)
				m.coolantFlowLMin += (rng.Float64()*2 - 1) * 0.1
				clampF(&m.coolantFlowLMin, 0.0, 5.0)
				m.materialRemovalCM3M += (rng.Float64()*2 - 1) * 5.0
				clampF(&m.materialRemovalCM3M, 50.0, 400.0)
			}
			m.Unlock()
		}
	}
}

func (m *CNCRouterMachine) GetTags() map[string]interface{} {
	m.RLock()
	defer m.RUnlock()
	tags := machine.BaseTagValues(m.BaseMachine)
	tags["spindle_speed_rpm"] = m.spindleSpeedRPM
	tags["x_position_mm"] = m.xPositionMM
	tags["y_position_mm"] = m.yPositionMM
	tags["z_position_mm"] = m.zPositionMM
	tags["cut_depth_mm"] = m.cutDepthMM
	tags["feed_rate_mm_min"] = m.feedRateMMMin
	tags["dust_extraction_m3_h"] = m.dustExtractionM3H
	tags["tool_wear_pct"] = m.toolWearPct
	tags["power_consumption_kw"] = m.powerConsumptionKW
	tags["vibration_mm_s"] = m.vibrationMMS
	tags["vacuum_hold_down_kpa"] = m.vacuumHoldDownKPa
	tags["surface_finish_um"] = m.surfaceFinishUM
	tags["tool_temp_c"] = m.toolTempC
	tags["coolant_flow_l_min"] = m.coolantFlowLMin
	tags["material_removal_rate_cm3_min"] = m.materialRemovalCM3M
	return tags
}

func (m *CNCRouterMachine) GetTagDefinitions() []machine.TagDefinition {
	defs := machine.BaseTags()
	defs = append(defs,
		machine.TagDefinition{Name: "spindle_speed_rpm", DisplayName: "Spindle Speed (RPM)", NodeID: 7, DataType: "Double", Unit: "RPM", Description: "Spindle speed"},
		machine.TagDefinition{Name: "x_position_mm", DisplayName: "X Position (mm)", NodeID: 8, DataType: "Double", Unit: "mm", Description: "X axis position"},
		machine.TagDefinition{Name: "y_position_mm", DisplayName: "Y Position (mm)", NodeID: 9, DataType: "Double", Unit: "mm", Description: "Y axis position"},
		machine.TagDefinition{Name: "z_position_mm", DisplayName: "Z Position (mm)", NodeID: 10, DataType: "Double", Unit: "mm", Description: "Z axis position"},
		machine.TagDefinition{Name: "cut_depth_mm", DisplayName: "Cut Depth (mm)", NodeID: 11, DataType: "Double", Unit: "mm", Description: "Cut depth"},
		machine.TagDefinition{Name: "feed_rate_mm_min", DisplayName: "Feed Rate (mm/min)", NodeID: 12, DataType: "Double", Unit: "mm/min", Description: "Feed rate"},
		machine.TagDefinition{Name: "dust_extraction_m3_h", DisplayName: "Dust Extraction (m\u00b3/h)", NodeID: 13, DataType: "Double", Unit: "m³/h", Description: "Dust extraction airflow"},
		machine.TagDefinition{Name: "tool_wear_pct", DisplayName: "Tool Wear (%)", NodeID: 14, DataType: "Double", Unit: "%", Description: "Tool wear percentage"},
		machine.TagDefinition{Name: "power_consumption_kw", DisplayName: "Power Consumption (kW)", NodeID: 15, DataType: "Double", Unit: "kW", Description: "Power consumption"},
		machine.TagDefinition{Name: "vibration_mm_s", DisplayName: "Vibration (mm/s)", NodeID: 16, DataType: "Double", Unit: "mm/s", Description: "Vibration level"},
		machine.TagDefinition{Name: "vacuum_hold_down_kpa", DisplayName: "Vacuum Hold-Down (kPa)", NodeID: 17, DataType: "Double", Unit: "kPa", Description: "Vacuum hold-down pressure"},
		machine.TagDefinition{Name: "surface_finish_um", DisplayName: "Surface Finish (\u00b5m)", NodeID: 18, DataType: "Double", Unit: "µm", Description: "Surface finish quality"},
		machine.TagDefinition{Name: "tool_temp_c", DisplayName: "Tool Temperature (\u00b0C)", NodeID: 19, DataType: "Double", Unit: "°C", Description: "Tool temperature"},
		machine.TagDefinition{Name: "coolant_flow_l_min", DisplayName: "Coolant Flow (L/min)", NodeID: 20, DataType: "Double", Unit: "L/min", Description: "Coolant flow rate"},
		machine.TagDefinition{Name: "material_removal_rate_cm3_min", DisplayName: "Material Removal Rate (cm\u00b3/min)", NodeID: 21, DataType: "Double", Unit: "cm³/min", Description: "Material removal rate"},
	)
	return defs
}
