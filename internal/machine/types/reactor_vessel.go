package types

import (
	"context"
	"time"

	"machine-simulator-2/internal/config"
	"machine-simulator-2/internal/machine"
)

func init() {
	machine.Register("reactor_vessel", func(cfg config.MachineInstanceConfig) machine.Machine {
		return NewReactorVessel(cfg)
	})
}

// ReactorVesselMachine simulates a pharmaceutical reactor vessel.
type ReactorVesselMachine struct {
	*machine.BaseMachine

	vesselTempC       float64
	vesselPressBar    float64
	agitatorSpeedRPM  float64
	phLevel           float64
	batchVolumeL      float64
	dissolvedO2PPM    float64
	conductivityUSCM  float64
	turbidityNTU      float64
	jacketTempC       float64
	feedRateMLMin     float64
	offGasFlowLMin    float64
	torqueNM          float64
	impellerTipMS     float64
	redoxPotentialMV  float64
	foamLevelPct      float64
}

func NewReactorVessel(cfg config.MachineInstanceConfig) *ReactorVesselMachine {
	return &ReactorVesselMachine{
		BaseMachine:      machine.NewBaseMachine(cfg),
		vesselTempC:      37.0,
		vesselPressBar:   1.2,
		agitatorSpeedRPM: 120.0,
		phLevel:          7.0,
		batchVolumeL:     500.0,
		dissolvedO2PPM:   6.0,
		conductivityUSCM: 1200.0,
		turbidityNTU:     15.0,
		jacketTempC:      35.0,
		feedRateMLMin:    50.0,
		offGasFlowLMin:   5.0,
		torqueNM:         25.0,
		impellerTipMS:    3.0,
		redoxPotentialMV: 200.0,
		foamLevelPct:     10.0,
	}
}

func (m *ReactorVesselMachine) Start(ctx context.Context) {
	go m.updateTags(ctx)
	m.BaseMachine.Start(ctx)
}

func (m *ReactorVesselMachine) updateTags(ctx context.Context) {
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
				m.vesselTempC += (rng.Float64()*2 - 1) * 0.1
				clampF(&m.vesselTempC, 20.0, 80.0)
				m.vesselPressBar += (rng.Float64()*2 - 1) * 0.02
				clampF(&m.vesselPressBar, 0.5, 5.0)
				m.agitatorSpeedRPM += (rng.Float64()*2 - 1) * 1.0
				clampF(&m.agitatorSpeedRPM, 30.0, 300.0)
				m.phLevel += (rng.Float64()*2 - 1) * 0.02
				clampF(&m.phLevel, 4.0, 10.0)
				m.batchVolumeL += (rng.Float64()*2 - 1) * 0.5
				clampF(&m.batchVolumeL, 100.0, 1000.0)
				m.dissolvedO2PPM += (rng.Float64()*2 - 1) * 0.1
				clampF(&m.dissolvedO2PPM, 0.5, 12.0)
				m.conductivityUSCM += (rng.Float64()*2 - 1) * 5.0
				clampF(&m.conductivityUSCM, 500.0, 3000.0)
				m.turbidityNTU += (rng.Float64()*2 - 1) * 0.5
				clampF(&m.turbidityNTU, 1.0, 100.0)
				m.jacketTempC += (rng.Float64()*2 - 1) * 0.1
				clampF(&m.jacketTempC, 15.0, 85.0)
				m.feedRateMLMin += (rng.Float64()*2 - 1) * 1.0
				clampF(&m.feedRateMLMin, 5.0, 200.0)
				m.offGasFlowLMin += (rng.Float64()*2 - 1) * 0.2
				clampF(&m.offGasFlowLMin, 0.5, 20.0)
				m.torqueNM += (rng.Float64()*2 - 1) * 0.5
				clampF(&m.torqueNM, 5.0, 80.0)
				m.impellerTipMS += (rng.Float64()*2 - 1) * 0.05
				clampF(&m.impellerTipMS, 1.0, 8.0)
				m.redoxPotentialMV += (rng.Float64()*2 - 1) * 2.0
				clampF(&m.redoxPotentialMV, -200.0, 500.0)
				m.foamLevelPct += (rng.Float64()*2 - 1) * 0.5
				clampF(&m.foamLevelPct, 0.0, 40.0)
			}
			m.Unlock()
		}
	}
}

func (m *ReactorVesselMachine) GetTags() map[string]interface{} {
	m.RLock()
	defer m.RUnlock()
	tags := machine.BaseTagValues(m.BaseMachine)
	tags["vessel_temp_c"] = m.vesselTempC
	tags["vessel_pressure_bar"] = m.vesselPressBar
	tags["agitator_speed_rpm"] = m.agitatorSpeedRPM
	tags["ph_level"] = m.phLevel
	tags["batch_volume_l"] = m.batchVolumeL
	tags["dissolved_oxygen_ppm"] = m.dissolvedO2PPM
	tags["conductivity_us_cm"] = m.conductivityUSCM
	tags["turbidity_ntu"] = m.turbidityNTU
	tags["jacket_temp_c"] = m.jacketTempC
	tags["feed_rate_ml_min"] = m.feedRateMLMin
	tags["off_gas_flow_l_min"] = m.offGasFlowLMin
	tags["torque_nm"] = m.torqueNM
	tags["impeller_tip_speed_m_s"] = m.impellerTipMS
	tags["redox_potential_mv"] = m.redoxPotentialMV
	tags["foam_level_pct"] = m.foamLevelPct
	return tags
}

func (m *ReactorVesselMachine) GetTagDefinitions() []machine.TagDefinition {
	defs := machine.BaseTags()
	defs = append(defs,
		machine.TagDefinition{Name: "vessel_temp_c", DisplayName: "Vessel Temp (°C)", NodeID: 7, DataType: "Double", Unit: "°C", Description: "Vessel temperature"},
		machine.TagDefinition{Name: "vessel_pressure_bar", DisplayName: "Vessel Pressure (bar)", NodeID: 8, DataType: "Double", Unit: "bar", Description: "Vessel pressure"},
		machine.TagDefinition{Name: "agitator_speed_rpm", DisplayName: "Agitator Speed (RPM)", NodeID: 9, DataType: "Double", Unit: "RPM", Description: "Agitator speed"},
		machine.TagDefinition{Name: "ph_level", DisplayName: "pH Level (pH)", NodeID: 10, DataType: "Double", Unit: "pH", Description: "pH level"},
		machine.TagDefinition{Name: "batch_volume_l", DisplayName: "Batch Volume (L)", NodeID: 11, DataType: "Double", Unit: "L", Description: "Batch volume"},
		machine.TagDefinition{Name: "dissolved_oxygen_ppm", DisplayName: "Dissolved Oxygen (ppm)", NodeID: 12, DataType: "Double", Unit: "ppm", Description: "Dissolved oxygen"},
		machine.TagDefinition{Name: "conductivity_us_cm", DisplayName: "Conductivity (µS/cm)", NodeID: 13, DataType: "Double", Unit: "µS/cm", Description: "Conductivity"},
		machine.TagDefinition{Name: "turbidity_ntu", DisplayName: "Turbidity (NTU)", NodeID: 14, DataType: "Double", Unit: "NTU", Description: "Turbidity"},
		machine.TagDefinition{Name: "jacket_temp_c", DisplayName: "Jacket Temp (°C)", NodeID: 15, DataType: "Double", Unit: "°C", Description: "Jacket temperature"},
		machine.TagDefinition{Name: "feed_rate_ml_min", DisplayName: "Feed Rate (mL/min)", NodeID: 16, DataType: "Double", Unit: "mL/min", Description: "Feed rate"},
		machine.TagDefinition{Name: "off_gas_flow_l_min", DisplayName: "Off-Gas Flow (L/min)", NodeID: 17, DataType: "Double", Unit: "L/min", Description: "Off-gas flow rate"},
		machine.TagDefinition{Name: "torque_nm", DisplayName: "Torque (N·m)", NodeID: 18, DataType: "Double", Unit: "N·m", Description: "Agitator torque"},
		machine.TagDefinition{Name: "impeller_tip_speed_m_s", DisplayName: "Impeller Tip Speed (m/s)", NodeID: 19, DataType: "Double", Unit: "m/s", Description: "Impeller tip speed"},
		machine.TagDefinition{Name: "redox_potential_mv", DisplayName: "Redox Potential (mV)", NodeID: 20, DataType: "Double", Unit: "mV", Description: "Redox potential"},
		machine.TagDefinition{Name: "foam_level_pct", DisplayName: "Foam Level (%)", NodeID: 21, DataType: "Double", Unit: "%", Description: "Foam level"},
	)
	return defs
}
