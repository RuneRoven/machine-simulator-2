package types

import (
	"context"
	"time"

	"machine-simulator-2/internal/config"
	"machine-simulator-2/internal/machine"
)

func init() {
	machine.Register("laser_cutter", func(cfg config.MachineInstanceConfig) machine.Machine {
		return NewLaserCutter(cfg)
	})
}

// LaserCutterMachine simulates a fiber/CO2 laser cutting machine.
type LaserCutterMachine struct {
	*machine.BaseMachine

	laserPowerW       float64
	cuttingSpeedMMS   float64
	assistGasPressBar float64
	focalPointMM      float64
	kerfWidthMM       float64
	beamQualityMMMrad float64
	nozzleStandoffMM  float64
	pierceTimeMS      float64
	sheetTempC        float64
	exhaustFlowM3H    float64
	lensConditionPct  float64
	gasConsumptLMin   float64
	edgeRoughnessUM   float64
	taperAngleDeg     float64
	powerStabilityPct float64
}

func NewLaserCutter(cfg config.MachineInstanceConfig) *LaserCutterMachine {
	return &LaserCutterMachine{
		BaseMachine:       machine.NewBaseMachine(cfg),
		laserPowerW:       4000.0,
		cuttingSpeedMMS:   40.0,
		assistGasPressBar: 12.0,
		focalPointMM:      0.0,
		kerfWidthMM:       0.2,
		beamQualityMMMrad: 4.0,
		nozzleStandoffMM:  1.0,
		pierceTimeMS:      200.0,
		sheetTempC:        30.0,
		exhaustFlowM3H:    3000.0,
		lensConditionPct:  95.0,
		gasConsumptLMin:   20.0,
		edgeRoughnessUM:   12.0,
		taperAngleDeg:     0.5,
		powerStabilityPct: 98.0,
	}
}

func (m *LaserCutterMachine) Start(ctx context.Context) {
	go m.updateTags(ctx)
	m.BaseMachine.Start(ctx)
}

func (m *LaserCutterMachine) updateTags(ctx context.Context) {
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
				m.laserPowerW += (rng.Float64()*2 - 1) * 20.0
				clampF(&m.laserPowerW, 500.0, 6000.0)
				m.cuttingSpeedMMS += (rng.Float64()*2 - 1) * 0.5
				clampF(&m.cuttingSpeedMMS, 5.0, 100.0)
				m.assistGasPressBar += (rng.Float64()*2 - 1) * 0.2
				clampF(&m.assistGasPressBar, 2.0, 25.0)
				m.focalPointMM += (rng.Float64()*2 - 1) * 0.02
				clampF(&m.focalPointMM, -3.0, 3.0)
				m.kerfWidthMM += (rng.Float64()*2 - 1) * 0.005
				clampF(&m.kerfWidthMM, 0.1, 0.5)
				m.beamQualityMMMrad += (rng.Float64()*2 - 1) * 0.05
				clampF(&m.beamQualityMMMrad, 2.0, 8.0)
				m.nozzleStandoffMM += (rng.Float64()*2 - 1) * 0.02
				clampF(&m.nozzleStandoffMM, 0.3, 3.0)
				m.pierceTimeMS += (rng.Float64()*2 - 1) * 5.0
				clampF(&m.pierceTimeMS, 50.0, 2000.0)
				m.sheetTempC += (rng.Float64()*2 - 1) * 0.5
				clampF(&m.sheetTempC, 20.0, 80.0)
				m.exhaustFlowM3H += (rng.Float64()*2 - 1) * 30.0
				clampF(&m.exhaustFlowM3H, 1500.0, 5000.0)
				m.lensConditionPct -= rng.Float64() * 0.001
				clampF(&m.lensConditionPct, 50.0, 100.0)
				m.gasConsumptLMin += (rng.Float64()*2 - 1) * 0.3
				clampF(&m.gasConsumptLMin, 5.0, 50.0)
				m.edgeRoughnessUM += (rng.Float64()*2 - 1) * 0.3
				clampF(&m.edgeRoughnessUM, 3.0, 30.0)
				m.taperAngleDeg += (rng.Float64()*2 - 1) * 0.02
				clampF(&m.taperAngleDeg, 0.0, 2.0)
				m.powerStabilityPct += (rng.Float64()*2 - 1) * 0.1
				clampF(&m.powerStabilityPct, 90.0, 100.0)
			}
			m.Unlock()
		}
	}
}

func (m *LaserCutterMachine) GetTags() map[string]interface{} {
	m.RLock()
	defer m.RUnlock()
	tags := machine.BaseTagValues(m.BaseMachine)
	tags["laser_power_w"] = m.laserPowerW
	tags["cutting_speed_mm_s"] = m.cuttingSpeedMMS
	tags["assist_gas_pressure_bar"] = m.assistGasPressBar
	tags["focal_point_mm"] = m.focalPointMM
	tags["kerf_width_mm"] = m.kerfWidthMM
	tags["beam_quality_mm_mrad"] = m.beamQualityMMMrad
	tags["nozzle_standoff_mm"] = m.nozzleStandoffMM
	tags["pierce_time_ms"] = m.pierceTimeMS
	tags["sheet_temp_c"] = m.sheetTempC
	tags["exhaust_flow_m3_h"] = m.exhaustFlowM3H
	tags["lens_condition_pct"] = m.lensConditionPct
	tags["gas_consumption_l_min"] = m.gasConsumptLMin
	tags["edge_roughness_um"] = m.edgeRoughnessUM
	tags["taper_angle_deg"] = m.taperAngleDeg
	tags["power_stability_pct"] = m.powerStabilityPct
	return tags
}

func (m *LaserCutterMachine) GetTagDefinitions() []machine.TagDefinition {
	defs := machine.BaseTags()
	defs = append(defs,
		machine.TagDefinition{Name: "laser_power_w", DisplayName: "Laser Power (W)", NodeID: 7, DataType: "Double", Unit: "W", Description: "Laser output power"},
		machine.TagDefinition{Name: "cutting_speed_mm_s", DisplayName: "Cutting Speed (mm/s)", NodeID: 8, DataType: "Double", Unit: "mm/s", Description: "Cutting speed"},
		machine.TagDefinition{Name: "assist_gas_pressure_bar", DisplayName: "Assist Gas Pressure (bar)", NodeID: 9, DataType: "Double", Unit: "bar", Description: "Assist gas pressure"},
		machine.TagDefinition{Name: "focal_point_mm", DisplayName: "Focal Point (mm)", NodeID: 10, DataType: "Double", Unit: "mm", Description: "Focal point position"},
		machine.TagDefinition{Name: "kerf_width_mm", DisplayName: "Kerf Width (mm)", NodeID: 11, DataType: "Double", Unit: "mm", Description: "Kerf width"},
		machine.TagDefinition{Name: "beam_quality_mm_mrad", DisplayName: "Beam Quality (mm\u00b7mrad)", NodeID: 12, DataType: "Double", Unit: "mm·mrad", Description: "Beam quality (BPP)"},
		machine.TagDefinition{Name: "nozzle_standoff_mm", DisplayName: "Nozzle Standoff (mm)", NodeID: 13, DataType: "Double", Unit: "mm", Description: "Nozzle standoff distance"},
		machine.TagDefinition{Name: "pierce_time_ms", DisplayName: "Pierce Time (ms)", NodeID: 14, DataType: "Double", Unit: "ms", Description: "Pierce time"},
		machine.TagDefinition{Name: "sheet_temp_c", DisplayName: "Sheet Temperature (\u00b0C)", NodeID: 15, DataType: "Double", Unit: "°C", Description: "Sheet temperature"},
		machine.TagDefinition{Name: "exhaust_flow_m3_h", DisplayName: "Exhaust Flow (m\u00b3/h)", NodeID: 16, DataType: "Double", Unit: "m³/h", Description: "Exhaust flow rate"},
		machine.TagDefinition{Name: "lens_condition_pct", DisplayName: "Lens Condition (%)", NodeID: 17, DataType: "Double", Unit: "%", Description: "Lens condition"},
		machine.TagDefinition{Name: "gas_consumption_l_min", DisplayName: "Gas Consumption (L/min)", NodeID: 18, DataType: "Double", Unit: "L/min", Description: "Gas consumption rate"},
		machine.TagDefinition{Name: "edge_roughness_um", DisplayName: "Edge Roughness (\u00b5m)", NodeID: 19, DataType: "Double", Unit: "µm", Description: "Cut edge roughness"},
		machine.TagDefinition{Name: "taper_angle_deg", DisplayName: "Taper Angle (\u00b0)", NodeID: 20, DataType: "Double", Unit: "°", Description: "Cut taper angle"},
		machine.TagDefinition{Name: "power_stability_pct", DisplayName: "Power Stability (%)", NodeID: 21, DataType: "Double", Unit: "%", Description: "Laser power stability"},
	)
	return defs
}
