package types

import (
	"context"
	"time"

	"machine-simulator-2/internal/config"
	"machine-simulator-2/internal/machine"
)

func init() {
	machine.Register("deburring_machine", func(cfg config.MachineInstanceConfig) machine.Machine {
		return NewDeburringMachine(cfg)
	})
}

// DeburringMachineMachine simulates a brush or rotary deburring machine.
type DeburringMachineMachine struct {
	*machine.BaseMachine

	brushSpeedRPM       float64
	contactPressureBar  float64
	feedRateMMS         float64
	surfaceFinishRaUM   float64
	materialRemovalMM   float64
	brushWearPct        float64
	powerConsumptionKW  float64
	vibrationMMS        float64
	coolantFlowLMin     float64
	dustExtractionM3H   float64
	partTempC           float64
	spindleCurrentA     float64
	burrHeightBeforeUM  float64
	burrHeightAfterUM   float64
	cycleCountBrush     float64
}

func NewDeburringMachine(cfg config.MachineInstanceConfig) *DeburringMachineMachine {
	return &DeburringMachineMachine{
		BaseMachine:        machine.NewBaseMachine(cfg),
		brushSpeedRPM:      1500.0,
		contactPressureBar: 2.0,
		feedRateMMS:        30.0,
		surfaceFinishRaUM:  1.6,
		materialRemovalMM:  0.05,
		brushWearPct:       5.0,
		powerConsumptionKW: 3.5,
		vibrationMMS:       1.0,
		coolantFlowLMin:    2.0,
		dustExtractionM3H:  500.0,
		partTempC:          32.0,
		spindleCurrentA:    8.0,
		burrHeightBeforeUM: 200.0,
		burrHeightAfterUM:  10.0,
		cycleCountBrush:    0.0,
	}
}

func (m *DeburringMachineMachine) Start(ctx context.Context) {
	go m.updateTags(ctx)
	m.BaseMachine.Start(ctx)
}

func (m *DeburringMachineMachine) updateTags(ctx context.Context) {
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
				m.brushSpeedRPM += (rng.Float64()*2 - 1) * 10.0
				clampF(&m.brushSpeedRPM, 500.0, 3000.0)
				m.contactPressureBar += (rng.Float64()*2 - 1) * 0.05
				clampF(&m.contactPressureBar, 0.5, 5.0)
				m.feedRateMMS += (rng.Float64()*2 - 1) * 0.5
				clampF(&m.feedRateMMS, 10.0, 80.0)
				m.surfaceFinishRaUM += (rng.Float64()*2 - 1) * 0.05
				clampF(&m.surfaceFinishRaUM, 0.4, 6.3)
				m.materialRemovalMM += (rng.Float64()*2 - 1) * 0.002
				clampF(&m.materialRemovalMM, 0.01, 0.2)
				m.brushWearPct += rng.Float64() * 0.002
				clampF(&m.brushWearPct, 0.0, 100.0)
				m.powerConsumptionKW += (rng.Float64()*2 - 1) * 0.1
				clampF(&m.powerConsumptionKW, 1.0, 8.0)
				m.vibrationMMS += (rng.Float64()*2 - 1) * 0.05
				clampF(&m.vibrationMMS, 0.2, 4.0)
				m.coolantFlowLMin += (rng.Float64()*2 - 1) * 0.05
				clampF(&m.coolantFlowLMin, 0.5, 6.0)
				m.dustExtractionM3H += (rng.Float64()*2 - 1) * 10.0
				clampF(&m.dustExtractionM3H, 200.0, 1000.0)
				m.partTempC += (rng.Float64()*2 - 1) * 0.3
				clampF(&m.partTempC, 20.0, 60.0)
				m.spindleCurrentA += (rng.Float64()*2 - 1) * 0.2
				clampF(&m.spindleCurrentA, 3.0, 20.0)
				m.burrHeightBeforeUM += (rng.Float64()*2 - 1) * 5.0
				clampF(&m.burrHeightBeforeUM, 50.0, 500.0)
				m.burrHeightAfterUM += (rng.Float64()*2 - 1) * 0.5
				clampF(&m.burrHeightAfterUM, 1.0, 50.0)
				m.cycleCountBrush += rng.Float64() * 0.3
			}
			m.Unlock()
		}
	}
}

func (m *DeburringMachineMachine) GetTags() map[string]interface{} {
	m.RLock()
	defer m.RUnlock()
	tags := machine.BaseTagValues(m.BaseMachine)
	tags["brush_speed_rpm"] = m.brushSpeedRPM
	tags["contact_pressure_bar"] = m.contactPressureBar
	tags["feed_rate_mm_s"] = m.feedRateMMS
	tags["surface_finish_ra_um"] = m.surfaceFinishRaUM
	tags["material_removal_mm"] = m.materialRemovalMM
	tags["brush_wear_pct"] = m.brushWearPct
	tags["power_consumption_kw"] = m.powerConsumptionKW
	tags["vibration_mm_s"] = m.vibrationMMS
	tags["coolant_flow_l_min"] = m.coolantFlowLMin
	tags["dust_extraction_m3_h"] = m.dustExtractionM3H
	tags["part_temp_c"] = m.partTempC
	tags["spindle_current_a"] = m.spindleCurrentA
	tags["burr_height_before_um"] = m.burrHeightBeforeUM
	tags["burr_height_after_um"] = m.burrHeightAfterUM
	tags["cycle_count_brush"] = m.cycleCountBrush
	return tags
}

func (m *DeburringMachineMachine) GetTagDefinitions() []machine.TagDefinition {
	defs := machine.BaseTags()
	defs = append(defs,
		machine.TagDefinition{Name: "brush_speed_rpm", DisplayName: "Brush Speed (RPM)", NodeID: 7, DataType: "Double", Unit: "RPM", Description: "Brush rotation speed"},
		machine.TagDefinition{Name: "contact_pressure_bar", DisplayName: "Contact Pressure (bar)", NodeID: 8, DataType: "Double", Unit: "bar", Description: "Contact pressure"},
		machine.TagDefinition{Name: "feed_rate_mm_s", DisplayName: "Feed Rate (mm/s)", NodeID: 9, DataType: "Double", Unit: "mm/s", Description: "Feed rate"},
		machine.TagDefinition{Name: "surface_finish_ra_um", DisplayName: "Surface Finish (\u00b5m Ra)", NodeID: 10, DataType: "Double", Unit: "µm Ra", Description: "Surface finish"},
		machine.TagDefinition{Name: "material_removal_mm", DisplayName: "Material Removal (mm)", NodeID: 11, DataType: "Double", Unit: "mm", Description: "Material removal depth"},
		machine.TagDefinition{Name: "brush_wear_pct", DisplayName: "Brush Wear (%)", NodeID: 12, DataType: "Double", Unit: "%", Description: "Brush wear percentage"},
		machine.TagDefinition{Name: "power_consumption_kw", DisplayName: "Power Consumption (kW)", NodeID: 13, DataType: "Double", Unit: "kW", Description: "Power consumption"},
		machine.TagDefinition{Name: "vibration_mm_s", DisplayName: "Vibration (mm/s)", NodeID: 14, DataType: "Double", Unit: "mm/s", Description: "Vibration level"},
		machine.TagDefinition{Name: "coolant_flow_l_min", DisplayName: "Coolant Flow (L/min)", NodeID: 15, DataType: "Double", Unit: "L/min", Description: "Coolant flow rate"},
		machine.TagDefinition{Name: "dust_extraction_m3_h", DisplayName: "Dust Extraction (m\u00b3/h)", NodeID: 16, DataType: "Double", Unit: "m³/h", Description: "Dust extraction airflow"},
		machine.TagDefinition{Name: "part_temp_c", DisplayName: "Part Temperature (\u00b0C)", NodeID: 17, DataType: "Double", Unit: "°C", Description: "Part temperature"},
		machine.TagDefinition{Name: "spindle_current_a", DisplayName: "Spindle Current (A)", NodeID: 18, DataType: "Double", Unit: "A", Description: "Spindle motor current"},
		machine.TagDefinition{Name: "burr_height_before_um", DisplayName: "Burr Height Before (\u00b5m)", NodeID: 19, DataType: "Double", Unit: "µm", Description: "Burr height before processing"},
		machine.TagDefinition{Name: "burr_height_after_um", DisplayName: "Burr Height After (\u00b5m)", NodeID: 20, DataType: "Double", Unit: "µm", Description: "Burr height after processing"},
		machine.TagDefinition{Name: "cycle_count_brush", DisplayName: "Brush Cycle Count", NodeID: 21, DataType: "Double", Unit: "", Description: "Brush cycle count"},
	)
	return defs
}
