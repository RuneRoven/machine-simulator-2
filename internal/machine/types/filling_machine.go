package types

import (
	"context"
	"time"

	"machine-simulator-2/internal/config"
	"machine-simulator-2/internal/machine"
)

func init() {
	machine.Register("filling_machine", func(cfg config.MachineInstanceConfig) machine.Machine {
		return NewFillingMachine(cfg)
	})
}

// FillingMachineMachine simulates a liquid filling machine for food and beverage.
type FillingMachineMachine struct {
	*machine.BaseMachine

	fillVolumeML     float64
	fillRateMLs      float64
	nozzlePressBar   float64
	productTempC     float64
	tankLevelPct     float64
	flowRateLMin     float64
	fillAccuracyPct  float64
	dripCount        float64
	valveCycleCount  float64
	cipStatus        float64
	productViscCP    float64
	linePressureBar  float64
	fillWeightG      float64
	densityKgM3      float64
	foamLevelPct     float64
}

func NewFillingMachine(cfg config.MachineInstanceConfig) *FillingMachineMachine {
	return &FillingMachineMachine{
		BaseMachine:     machine.NewBaseMachine(cfg),
		fillVolumeML:    500.0,
		fillRateMLs:     120.0,
		nozzlePressBar:  2.5,
		productTempC:    4.0,
		tankLevelPct:    85.0,
		flowRateLMin:    7.2,
		fillAccuracyPct: 99.5,
		dripCount:       0.0,
		valveCycleCount: 0.0,
		cipStatus:       0.0,
		productViscCP:   3.0,
		linePressureBar: 3.0,
		fillWeightG:     505.0,
		densityKgM3:     1010.0,
		foamLevelPct:    5.0,
	}
}

func (m *FillingMachineMachine) Start(ctx context.Context) {
	go m.updateTags(ctx)
	m.BaseMachine.Start(ctx)
}

func (m *FillingMachineMachine) updateTags(ctx context.Context) {
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
				m.fillVolumeML += (rng.Float64()*2 - 1) * 1.0
				clampF(&m.fillVolumeML, 450.0, 550.0)
				m.fillRateMLs += (rng.Float64()*2 - 1) * 2.0
				clampF(&m.fillRateMLs, 80.0, 200.0)
				m.nozzlePressBar += (rng.Float64()*2 - 1) * 0.05
				clampF(&m.nozzlePressBar, 1.5, 4.0)
				m.productTempC += (rng.Float64()*2 - 1) * 0.1
				clampF(&m.productTempC, 1.0, 10.0)
				m.tankLevelPct -= rng.Float64() * 0.05
				clampF(&m.tankLevelPct, 10.0, 100.0)
				m.flowRateLMin += (rng.Float64()*2 - 1) * 0.1
				clampF(&m.flowRateLMin, 3.0, 15.0)
				m.fillAccuracyPct += (rng.Float64()*2 - 1) * 0.05
				clampF(&m.fillAccuracyPct, 97.0, 100.0)
				if rng.Float64() < 0.003 {
					m.dripCount++
				}
				m.valveCycleCount += rng.Float64() * 0.5
				m.productViscCP += (rng.Float64()*2 - 1) * 0.05
				clampF(&m.productViscCP, 1.0, 10.0)
				m.linePressureBar += (rng.Float64()*2 - 1) * 0.05
				clampF(&m.linePressureBar, 1.5, 5.0)
				m.fillWeightG += (rng.Float64()*2 - 1) * 1.0
				clampF(&m.fillWeightG, 450.0, 560.0)
				m.densityKgM3 += (rng.Float64()*2 - 1) * 0.5
				clampF(&m.densityKgM3, 990.0, 1050.0)
				m.foamLevelPct += (rng.Float64()*2 - 1) * 0.3
				clampF(&m.foamLevelPct, 0.0, 20.0)
			}
			m.Unlock()
		}
	}
}

func (m *FillingMachineMachine) GetTags() map[string]interface{} {
	m.RLock()
	defer m.RUnlock()
	tags := machine.BaseTagValues(m.BaseMachine)
	tags["fill_volume_ml"] = m.fillVolumeML
	tags["fill_rate_ml_s"] = m.fillRateMLs
	tags["nozzle_pressure_bar"] = m.nozzlePressBar
	tags["product_temp_c"] = m.productTempC
	tags["tank_level_pct"] = m.tankLevelPct
	tags["flow_rate_l_min"] = m.flowRateLMin
	tags["fill_accuracy_pct"] = m.fillAccuracyPct
	tags["drip_count"] = m.dripCount
	tags["valve_cycle_count"] = m.valveCycleCount
	tags["cip_status"] = m.cipStatus
	tags["product_viscosity_cp"] = m.productViscCP
	tags["line_pressure_bar"] = m.linePressureBar
	tags["fill_weight_g"] = m.fillWeightG
	tags["density_kg_m3"] = m.densityKgM3
	tags["foam_level_pct"] = m.foamLevelPct
	return tags
}

func (m *FillingMachineMachine) GetTagDefinitions() []machine.TagDefinition {
	defs := machine.BaseTags()
	defs = append(defs,
		machine.TagDefinition{Name: "fill_volume_ml", DisplayName: "Fill Volume (mL)", NodeID: 7, DataType: "Double", Unit: "mL", Description: "Fill volume per container"},
		machine.TagDefinition{Name: "fill_rate_ml_s", DisplayName: "Fill Rate (mL/s)", NodeID: 8, DataType: "Double", Unit: "mL/s", Description: "Fill rate"},
		machine.TagDefinition{Name: "nozzle_pressure_bar", DisplayName: "Nozzle Pressure (bar)", NodeID: 9, DataType: "Double", Unit: "bar", Description: "Nozzle pressure"},
		machine.TagDefinition{Name: "product_temp_c", DisplayName: "Product Temp (°C)", NodeID: 10, DataType: "Double", Unit: "°C", Description: "Product temperature"},
		machine.TagDefinition{Name: "tank_level_pct", DisplayName: "Tank Level (%)", NodeID: 11, DataType: "Double", Unit: "%", Description: "Supply tank level"},
		machine.TagDefinition{Name: "flow_rate_l_min", DisplayName: "Flow Rate (L/min)", NodeID: 12, DataType: "Double", Unit: "L/min", Description: "Product flow rate"},
		machine.TagDefinition{Name: "fill_accuracy_pct", DisplayName: "Fill Accuracy (%)", NodeID: 13, DataType: "Double", Unit: "%", Description: "Fill accuracy"},
		machine.TagDefinition{Name: "drip_count", DisplayName: "Drip Count", NodeID: 14, DataType: "Double", Unit: "", Description: "Nozzle drip events"},
		machine.TagDefinition{Name: "valve_cycle_count", DisplayName: "Valve Cycle Count", NodeID: 15, DataType: "Double", Unit: "", Description: "Fill valve cycles"},
		machine.TagDefinition{Name: "cip_status", DisplayName: "CIP Status", NodeID: 16, DataType: "Double", Unit: "", Description: "Clean-in-place status (0=idle, 1=active)"},
		machine.TagDefinition{Name: "product_viscosity_cp", DisplayName: "Product Viscosity (cP)", NodeID: 17, DataType: "Double", Unit: "cP", Description: "Product viscosity"},
		machine.TagDefinition{Name: "line_pressure_bar", DisplayName: "Line Pressure (bar)", NodeID: 18, DataType: "Double", Unit: "bar", Description: "Line pressure"},
		machine.TagDefinition{Name: "fill_weight_g", DisplayName: "Fill Weight (g)", NodeID: 19, DataType: "Double", Unit: "g", Description: "Measured fill weight"},
		machine.TagDefinition{Name: "density_kg_m3", DisplayName: "Density (kg/m³)", NodeID: 20, DataType: "Double", Unit: "kg/m³", Description: "Product density"},
		machine.TagDefinition{Name: "foam_level_pct", DisplayName: "Foam Level (%)", NodeID: 21, DataType: "Double", Unit: "%", Description: "Foam level in container"},
	)
	return defs
}
