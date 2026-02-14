package types

import (
	"context"
	"time"

	"machine-simulator-2/internal/config"
	"machine-simulator-2/internal/machine"
)

func init() {
	machine.Register("press_brake", func(cfg config.MachineInstanceConfig) machine.Machine {
		return NewPressBrake(cfg)
	})
}

// PressBrakeMachine simulates a hydraulic press brake for sheet metal bending.
type PressBrakeMachine struct {
	*machine.BaseMachine

	ramForceTons       float64
	bendAngleDeg       float64
	backGaugePosMM     float64
	strokeLengthMM     float64
	dieTempC           float64
	crowningMM         float64
	daylightMM         float64
	ramSpeedMMS        float64
	tonnagePct         float64
	springbackDeg      float64
	repeatabilityMM    float64
	hydraulicPressBar  float64
	beamDeflectionMM   float64
	energyConsKWH      float64
	cycleTimeActualS   float64
}

func NewPressBrake(cfg config.MachineInstanceConfig) *PressBrakeMachine {
	return &PressBrakeMachine{
		BaseMachine:       machine.NewBaseMachine(cfg),
		ramForceTons:      100.0,
		bendAngleDeg:      90.0,
		backGaugePosMM:    50.0,
		strokeLengthMM:    120.0,
		dieTempC:          30.0,
		crowningMM:        0.05,
		daylightMM:        400.0,
		ramSpeedMMS:       10.0,
		tonnagePct:        65.0,
		springbackDeg:     2.0,
		repeatabilityMM:   0.02,
		hydraulicPressBar: 200.0,
		beamDeflectionMM:  0.1,
		energyConsKWH:     1.5,
		cycleTimeActualS:  8.0,
	}
}

func (m *PressBrakeMachine) Start(ctx context.Context) {
	go m.updateTags(ctx)
	m.BaseMachine.Start(ctx)
}

func (m *PressBrakeMachine) updateTags(ctx context.Context) {
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
				m.ramForceTons += (rng.Float64()*2 - 1) * 2.0
				clampF(&m.ramForceTons, 10.0, 300.0)
				m.bendAngleDeg += (rng.Float64()*2 - 1) * 0.1
				clampF(&m.bendAngleDeg, 30.0, 175.0)
				m.backGaugePosMM += (rng.Float64()*2 - 1) * 0.5
				clampF(&m.backGaugePosMM, 5.0, 500.0)
				m.strokeLengthMM += (rng.Float64()*2 - 1) * 0.5
				clampF(&m.strokeLengthMM, 50.0, 300.0)
				m.dieTempC += (rng.Float64()*2 - 1) * 0.2
				clampF(&m.dieTempC, 20.0, 60.0)
				m.crowningMM += (rng.Float64()*2 - 1) * 0.005
				clampF(&m.crowningMM, 0.0, 0.3)
				m.daylightMM += (rng.Float64()*2 - 1) * 1.0
				clampF(&m.daylightMM, 200.0, 600.0)
				m.ramSpeedMMS += (rng.Float64()*2 - 1) * 0.2
				clampF(&m.ramSpeedMMS, 1.0, 25.0)
				m.tonnagePct += (rng.Float64()*2 - 1) * 1.0
				clampF(&m.tonnagePct, 20.0, 100.0)
				m.springbackDeg += (rng.Float64()*2 - 1) * 0.05
				clampF(&m.springbackDeg, 0.5, 5.0)
				m.repeatabilityMM += (rng.Float64()*2 - 1) * 0.002
				clampF(&m.repeatabilityMM, 0.005, 0.1)
				m.hydraulicPressBar += (rng.Float64()*2 - 1) * 2.0
				clampF(&m.hydraulicPressBar, 100.0, 350.0)
				m.beamDeflectionMM += (rng.Float64()*2 - 1) * 0.005
				clampF(&m.beamDeflectionMM, 0.0, 0.5)
				m.energyConsKWH += (rng.Float64()*2 - 1) * 0.05
				clampF(&m.energyConsKWH, 0.5, 5.0)
				m.cycleTimeActualS += (rng.Float64()*2 - 1) * 0.1
				clampF(&m.cycleTimeActualS, 3.0, 20.0)
			}
			m.Unlock()
		}
	}
}

func (m *PressBrakeMachine) GetTags() map[string]interface{} {
	m.RLock()
	defer m.RUnlock()
	tags := machine.BaseTagValues(m.BaseMachine)
	tags["ram_force_tons"] = m.ramForceTons
	tags["bend_angle_deg"] = m.bendAngleDeg
	tags["back_gauge_position_mm"] = m.backGaugePosMM
	tags["stroke_length_mm"] = m.strokeLengthMM
	tags["die_temp_c"] = m.dieTempC
	tags["crowning_mm"] = m.crowningMM
	tags["daylight_mm"] = m.daylightMM
	tags["ram_speed_mm_s"] = m.ramSpeedMMS
	tags["tonnage_pct"] = m.tonnagePct
	tags["material_springback_deg"] = m.springbackDeg
	tags["repeatability_mm"] = m.repeatabilityMM
	tags["hydraulic_pressure_bar"] = m.hydraulicPressBar
	tags["beam_deflection_mm"] = m.beamDeflectionMM
	tags["energy_consumption_kwh"] = m.energyConsKWH
	tags["cycle_time_actual_s"] = m.cycleTimeActualS
	return tags
}

func (m *PressBrakeMachine) GetTagDefinitions() []machine.TagDefinition {
	defs := machine.BaseTags()
	defs = append(defs,
		machine.TagDefinition{Name: "ram_force_tons", DisplayName: "Ram Force (tons)", NodeID: 7, DataType: "Double", Unit: "tons", Description: "Ram force"},
		machine.TagDefinition{Name: "bend_angle_deg", DisplayName: "Bend Angle (\u00b0)", NodeID: 8, DataType: "Double", Unit: "°", Description: "Bend angle"},
		machine.TagDefinition{Name: "back_gauge_position_mm", DisplayName: "Back Gauge Position (mm)", NodeID: 9, DataType: "Double", Unit: "mm", Description: "Back gauge position"},
		machine.TagDefinition{Name: "stroke_length_mm", DisplayName: "Stroke Length (mm)", NodeID: 10, DataType: "Double", Unit: "mm", Description: "Stroke length"},
		machine.TagDefinition{Name: "die_temp_c", DisplayName: "Die Temperature (\u00b0C)", NodeID: 11, DataType: "Double", Unit: "°C", Description: "Die temperature"},
		machine.TagDefinition{Name: "crowning_mm", DisplayName: "Crowning (mm)", NodeID: 12, DataType: "Double", Unit: "mm", Description: "Crowning compensation"},
		machine.TagDefinition{Name: "daylight_mm", DisplayName: "Daylight Opening (mm)", NodeID: 13, DataType: "Double", Unit: "mm", Description: "Daylight opening"},
		machine.TagDefinition{Name: "ram_speed_mm_s", DisplayName: "Ram Speed (mm/s)", NodeID: 14, DataType: "Double", Unit: "mm/s", Description: "Ram speed"},
		machine.TagDefinition{Name: "tonnage_pct", DisplayName: "Tonnage Utilization (%)", NodeID: 15, DataType: "Double", Unit: "%", Description: "Tonnage utilization"},
		machine.TagDefinition{Name: "material_springback_deg", DisplayName: "Springback (\u00b0)", NodeID: 16, DataType: "Double", Unit: "°", Description: "Material springback"},
		machine.TagDefinition{Name: "repeatability_mm", DisplayName: "Repeatability (mm)", NodeID: 17, DataType: "Double", Unit: "mm", Description: "Position repeatability"},
		machine.TagDefinition{Name: "hydraulic_pressure_bar", DisplayName: "Hydraulic Pressure (bar)", NodeID: 18, DataType: "Double", Unit: "bar", Description: "Hydraulic pressure"},
		machine.TagDefinition{Name: "beam_deflection_mm", DisplayName: "Beam Deflection (mm)", NodeID: 19, DataType: "Double", Unit: "mm", Description: "Beam deflection"},
		machine.TagDefinition{Name: "energy_consumption_kwh", DisplayName: "Energy Consumption (kWh)", NodeID: 20, DataType: "Double", Unit: "kWh", Description: "Energy consumption"},
		machine.TagDefinition{Name: "cycle_time_actual_s", DisplayName: "Actual Cycle Time (s)", NodeID: 21, DataType: "Double", Unit: "s", Description: "Actual cycle time"},
	)
	return defs
}
