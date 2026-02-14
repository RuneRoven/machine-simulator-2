package types

import (
	"context"
	"time"

	"machine-simulator-2/internal/config"
	"machine-simulator-2/internal/machine"
)

func init() {
	machine.Register("painting_booth", func(cfg config.MachineInstanceConfig) machine.Machine {
		return NewPaintingBooth(cfg)
	})
}

// PaintingBoothMachine simulates an automotive paint spray booth.
type PaintingBoothMachine struct {
	*machine.BaseMachine

	boothTempC        float64
	boothHumidityPct  float64
	paintFlowMLMin    float64
	atomizationBar    float64
	filmThicknessUM   float64
	oversprayPct      float64
	fanSpeedRPM       float64
	filterPressDropPa float64
	solventConcPPM    float64
	cureTempC         float64
	conveyorSpeedMMin float64
	paintViscosityCP  float64
	colorDeltaE       float64
	boothAirflowM3H   float64
	paintWastePct     float64
}

func NewPaintingBooth(cfg config.MachineInstanceConfig) *PaintingBoothMachine {
	return &PaintingBoothMachine{
		BaseMachine:       machine.NewBaseMachine(cfg),
		boothTempC:        22.0,
		boothHumidityPct:  55.0,
		paintFlowMLMin:    250.0,
		atomizationBar:    4.5,
		filmThicknessUM:   35.0,
		oversprayPct:      25.0,
		fanSpeedRPM:       1800.0,
		filterPressDropPa: 120.0,
		solventConcPPM:    350.0,
		cureTempC:         140.0,
		conveyorSpeedMMin: 3.5,
		paintViscosityCP:  85.0,
		colorDeltaE:       0.5,
		boothAirflowM3H:   25000.0,
		paintWastePct:     8.0,
	}
}

func (m *PaintingBoothMachine) Start(ctx context.Context) {
	go m.updateTags(ctx)
	m.BaseMachine.Start(ctx)
}

func (m *PaintingBoothMachine) updateTags(ctx context.Context) {
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
				m.boothTempC += (rng.Float64()*2 - 1) * 0.2
				clampF(&m.boothTempC, 18.0, 28.0)
				m.boothHumidityPct += (rng.Float64()*2 - 1) * 0.5
				clampF(&m.boothHumidityPct, 40.0, 70.0)
				m.paintFlowMLMin += (rng.Float64()*2 - 1) * 5.0
				clampF(&m.paintFlowMLMin, 100.0, 500.0)
				m.atomizationBar += (rng.Float64()*2 - 1) * 0.1
				clampF(&m.atomizationBar, 2.0, 7.0)
				m.filmThicknessUM += (rng.Float64()*2 - 1) * 1.0
				clampF(&m.filmThicknessUM, 15.0, 60.0)
				m.oversprayPct += (rng.Float64()*2 - 1) * 0.5
				clampF(&m.oversprayPct, 10.0, 45.0)
				m.fanSpeedRPM += (rng.Float64()*2 - 1) * 10.0
				clampF(&m.fanSpeedRPM, 1200.0, 2400.0)
				m.filterPressDropPa += rng.Float64() * 0.2
				clampF(&m.filterPressDropPa, 50.0, 500.0)
				m.solventConcPPM += (rng.Float64()*2 - 1) * 5.0
				clampF(&m.solventConcPPM, 100.0, 800.0)
				m.cureTempC += (rng.Float64()*2 - 1) * 0.5
				clampF(&m.cureTempC, 120.0, 180.0)
				m.conveyorSpeedMMin += (rng.Float64()*2 - 1) * 0.05
				clampF(&m.conveyorSpeedMMin, 1.0, 6.0)
				m.paintViscosityCP += (rng.Float64()*2 - 1) * 1.0
				clampF(&m.paintViscosityCP, 50.0, 150.0)
				m.colorDeltaE += (rng.Float64()*2 - 1) * 0.05
				clampF(&m.colorDeltaE, 0.0, 3.0)
				m.boothAirflowM3H += (rng.Float64()*2 - 1) * 100.0
				clampF(&m.boothAirflowM3H, 15000.0, 35000.0)
				m.paintWastePct += (rng.Float64()*2 - 1) * 0.2
				clampF(&m.paintWastePct, 2.0, 20.0)
			}
			m.Unlock()
		}
	}
}

func (m *PaintingBoothMachine) GetTags() map[string]interface{} {
	m.RLock()
	defer m.RUnlock()
	tags := machine.BaseTagValues(m.BaseMachine)
	tags["booth_temp_c"] = m.boothTempC
	tags["booth_humidity_pct"] = m.boothHumidityPct
	tags["paint_flow_ml_min"] = m.paintFlowMLMin
	tags["atomization_pressure_bar"] = m.atomizationBar
	tags["film_thickness_um"] = m.filmThicknessUM
	tags["overspray_pct"] = m.oversprayPct
	tags["fan_speed_rpm"] = m.fanSpeedRPM
	tags["filter_pressure_drop_pa"] = m.filterPressDropPa
	tags["solvent_concentration_ppm"] = m.solventConcPPM
	tags["cure_temp_c"] = m.cureTempC
	tags["conveyor_speed_m_min"] = m.conveyorSpeedMMin
	tags["paint_viscosity_cp"] = m.paintViscosityCP
	tags["color_delta_e"] = m.colorDeltaE
	tags["booth_airflow_m3_h"] = m.boothAirflowM3H
	tags["paint_waste_pct"] = m.paintWastePct
	return tags
}

func (m *PaintingBoothMachine) GetTagDefinitions() []machine.TagDefinition {
	defs := machine.BaseTags()
	defs = append(defs,
		machine.TagDefinition{Name: "booth_temp_c", DisplayName: "Booth Temperature (\u00b0C)", NodeID: 7, DataType: "Double", Unit: "°C", Description: "Booth temperature"},
		machine.TagDefinition{Name: "booth_humidity_pct", DisplayName: "Booth Humidity (%)", NodeID: 8, DataType: "Double", Unit: "%", Description: "Booth humidity"},
		machine.TagDefinition{Name: "paint_flow_ml_min", DisplayName: "Paint Flow Rate (mL/min)", NodeID: 9, DataType: "Double", Unit: "mL/min", Description: "Paint flow rate"},
		machine.TagDefinition{Name: "atomization_pressure_bar", DisplayName: "Atomization Pressure (bar)", NodeID: 10, DataType: "Double", Unit: "bar", Description: "Atomization air pressure"},
		machine.TagDefinition{Name: "film_thickness_um", DisplayName: "Film Thickness (\u00b5m)", NodeID: 11, DataType: "Double", Unit: "µm", Description: "Paint film thickness"},
		machine.TagDefinition{Name: "overspray_pct", DisplayName: "Overspray (%)", NodeID: 12, DataType: "Double", Unit: "%", Description: "Overspray percentage"},
		machine.TagDefinition{Name: "fan_speed_rpm", DisplayName: "Fan Speed (RPM)", NodeID: 13, DataType: "Double", Unit: "RPM", Description: "Exhaust fan speed"},
		machine.TagDefinition{Name: "filter_pressure_drop_pa", DisplayName: "Filter Pressure Drop (Pa)", NodeID: 14, DataType: "Double", Unit: "Pa", Description: "Filter pressure drop"},
		machine.TagDefinition{Name: "solvent_concentration_ppm", DisplayName: "Solvent Concentration (ppm)", NodeID: 15, DataType: "Double", Unit: "ppm", Description: "Solvent vapor concentration"},
		machine.TagDefinition{Name: "cure_temp_c", DisplayName: "Cure Oven Temp (\u00b0C)", NodeID: 16, DataType: "Double", Unit: "°C", Description: "Cure oven temperature"},
		machine.TagDefinition{Name: "conveyor_speed_m_min", DisplayName: "Conveyor Speed (m/min)", NodeID: 17, DataType: "Double", Unit: "m/min", Description: "Conveyor speed"},
		machine.TagDefinition{Name: "paint_viscosity_cp", DisplayName: "Paint Viscosity (cP)", NodeID: 18, DataType: "Double", Unit: "cP", Description: "Paint viscosity"},
		machine.TagDefinition{Name: "color_delta_e", DisplayName: "Color Deviation (\u0394E)", NodeID: 19, DataType: "Double", Unit: "ΔE", Description: "Color deviation"},
		machine.TagDefinition{Name: "booth_airflow_m3_h", DisplayName: "Booth Airflow (m\u00b3/h)", NodeID: 20, DataType: "Double", Unit: "m³/h", Description: "Booth airflow volume"},
		machine.TagDefinition{Name: "paint_waste_pct", DisplayName: "Paint Waste (%)", NodeID: 21, DataType: "Double", Unit: "%", Description: "Paint waste percentage"},
	)
	return defs
}
