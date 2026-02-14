package types

import (
	"context"
	"time"

	"machine-simulator-2/internal/config"
	"machine-simulator-2/internal/machine"
)

func init() {
	machine.Register("extruder", func(cfg config.MachineInstanceConfig) machine.Machine {
		return NewExtruder(cfg)
	})
}

// ExtruderMachine simulates a plastic extrusion machine.
type ExtruderMachine struct {
	*machine.BaseMachine

	zone1TempC        float64
	zone2TempC        float64
	zone3TempC        float64
	zone4TempC        float64
	screwSpeedRPM     float64
	meltPressureBar   float64
	diePressureBar    float64
	haulOffSpeedMMin  float64
	meltTempC         float64
	motorCurrentA     float64
	throughputKgH     float64
	torquePct         float64
	headPressureBar   float64
	vacuumMbar        float64
	coolingWaterTempC float64
	lineSpeedMMin     float64
}

func NewExtruder(cfg config.MachineInstanceConfig) *ExtruderMachine {
	return &ExtruderMachine{
		BaseMachine:       machine.NewBaseMachine(cfg),
		zone1TempC:        170.0,
		zone2TempC:        185.0,
		zone3TempC:        195.0,
		zone4TempC:        200.0,
		screwSpeedRPM:     80.0,
		meltPressureBar:   180.0,
		diePressureBar:    120.0,
		haulOffSpeedMMin:  5.0,
		meltTempC:         205.0,
		motorCurrentA:     45.0,
		throughputKgH:     60.0,
		torquePct:         70.0,
		headPressureBar:   150.0,
		vacuumMbar:        -200.0,
		coolingWaterTempC: 15.0,
		lineSpeedMMin:     5.0,
	}
}

func (m *ExtruderMachine) Start(ctx context.Context) {
	go m.updateTags(ctx)
	m.BaseMachine.Start(ctx)
}

func (m *ExtruderMachine) updateTags(ctx context.Context) {
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
				m.zone1TempC += (rng.Float64()*2 - 1) * 0.3
				clampF(&m.zone1TempC, 150.0, 200.0)
				m.zone2TempC += (rng.Float64()*2 - 1) * 0.3
				clampF(&m.zone2TempC, 165.0, 210.0)
				m.zone3TempC += (rng.Float64()*2 - 1) * 0.3
				clampF(&m.zone3TempC, 175.0, 220.0)
				m.zone4TempC += (rng.Float64()*2 - 1) * 0.3
				clampF(&m.zone4TempC, 180.0, 230.0)
				m.screwSpeedRPM += (rng.Float64()*2 - 1) * 0.5
				clampF(&m.screwSpeedRPM, 20.0, 150.0)
				m.meltPressureBar += (rng.Float64()*2 - 1) * 2.0
				clampF(&m.meltPressureBar, 80.0, 350.0)
				m.diePressureBar += (rng.Float64()*2 - 1) * 1.5
				clampF(&m.diePressureBar, 40.0, 250.0)
				m.haulOffSpeedMMin += (rng.Float64()*2 - 1) * 0.05
				clampF(&m.haulOffSpeedMMin, 1.0, 15.0)
				m.meltTempC += (rng.Float64()*2 - 1) * 0.3
				clampF(&m.meltTempC, 180.0, 260.0)
				m.motorCurrentA += (rng.Float64()*2 - 1) * 0.5
				clampF(&m.motorCurrentA, 15.0, 100.0)
				m.throughputKgH += (rng.Float64()*2 - 1) * 0.5
				clampF(&m.throughputKgH, 20.0, 150.0)
				m.torquePct += (rng.Float64()*2 - 1) * 0.5
				clampF(&m.torquePct, 30.0, 95.0)
				m.headPressureBar += (rng.Float64()*2 - 1) * 2.0
				clampF(&m.headPressureBar, 60.0, 300.0)
				m.vacuumMbar += (rng.Float64()*2 - 1) * 2.0
				clampF(&m.vacuumMbar, -500.0, -50.0)
				m.coolingWaterTempC += (rng.Float64()*2 - 1) * 0.1
				clampF(&m.coolingWaterTempC, 8.0, 25.0)
				m.lineSpeedMMin += (rng.Float64()*2 - 1) * 0.05
				clampF(&m.lineSpeedMMin, 1.0, 15.0)
			}
			m.Unlock()
		}
	}
}

func (m *ExtruderMachine) GetTags() map[string]interface{} {
	m.RLock()
	defer m.RUnlock()
	tags := machine.BaseTagValues(m.BaseMachine)
	tags["zone1_temp_c"] = m.zone1TempC
	tags["zone2_temp_c"] = m.zone2TempC
	tags["zone3_temp_c"] = m.zone3TempC
	tags["zone4_temp_c"] = m.zone4TempC
	tags["screw_speed_rpm"] = m.screwSpeedRPM
	tags["melt_pressure_bar"] = m.meltPressureBar
	tags["die_pressure_bar"] = m.diePressureBar
	tags["haul_off_speed_m_min"] = m.haulOffSpeedMMin
	tags["melt_temp_c"] = m.meltTempC
	tags["motor_current_a"] = m.motorCurrentA
	tags["throughput_kg_h"] = m.throughputKgH
	tags["torque_pct"] = m.torquePct
	tags["head_pressure_bar"] = m.headPressureBar
	tags["vacuum_mbar"] = m.vacuumMbar
	tags["cooling_water_temp_c"] = m.coolingWaterTempC
	tags["line_speed_m_min"] = m.lineSpeedMMin
	return tags
}

func (m *ExtruderMachine) GetTagDefinitions() []machine.TagDefinition {
	defs := machine.BaseTags()
	defs = append(defs,
		machine.TagDefinition{Name: "zone1_temp_c", DisplayName: "Zone 1 Temp (\u00b0C)", NodeID: 7, DataType: "Double", Unit: "°C", Description: "Barrel zone 1 temperature (feed)"},
		machine.TagDefinition{Name: "zone2_temp_c", DisplayName: "Zone 2 Temp (\u00b0C)", NodeID: 8, DataType: "Double", Unit: "°C", Description: "Barrel zone 2 temperature"},
		machine.TagDefinition{Name: "zone3_temp_c", DisplayName: "Zone 3 Temp (\u00b0C)", NodeID: 9, DataType: "Double", Unit: "°C", Description: "Barrel zone 3 temperature"},
		machine.TagDefinition{Name: "zone4_temp_c", DisplayName: "Zone 4 Temp (\u00b0C)", NodeID: 10, DataType: "Double", Unit: "°C", Description: "Barrel zone 4 temperature (die)"},
		machine.TagDefinition{Name: "screw_speed_rpm", DisplayName: "Screw Speed (RPM)", NodeID: 11, DataType: "Double", Unit: "RPM", Description: "Screw rotation speed"},
		machine.TagDefinition{Name: "melt_pressure_bar", DisplayName: "Melt Pressure (bar)", NodeID: 12, DataType: "Double", Unit: "bar", Description: "Melt pressure"},
		machine.TagDefinition{Name: "die_pressure_bar", DisplayName: "Die Pressure (bar)", NodeID: 13, DataType: "Double", Unit: "bar", Description: "Die pressure"},
		machine.TagDefinition{Name: "haul_off_speed_m_min", DisplayName: "Haul-Off Speed (m/min)", NodeID: 14, DataType: "Double", Unit: "m/min", Description: "Haul-off speed"},
		machine.TagDefinition{Name: "melt_temp_c", DisplayName: "Melt Temperature (\u00b0C)", NodeID: 15, DataType: "Double", Unit: "°C", Description: "Melt temperature"},
		machine.TagDefinition{Name: "motor_current_a", DisplayName: "Motor Current (A)", NodeID: 16, DataType: "Double", Unit: "A", Description: "Motor current draw"},
		machine.TagDefinition{Name: "throughput_kg_h", DisplayName: "Throughput (kg/h)", NodeID: 17, DataType: "Double", Unit: "kg/h", Description: "Material throughput"},
		machine.TagDefinition{Name: "torque_pct", DisplayName: "Torque Utilization (%)", NodeID: 18, DataType: "Double", Unit: "%", Description: "Screw torque utilization"},
		machine.TagDefinition{Name: "head_pressure_bar", DisplayName: "Head Pressure (bar)", NodeID: 19, DataType: "Double", Unit: "bar", Description: "Head pressure"},
		machine.TagDefinition{Name: "vacuum_mbar", DisplayName: "Vacuum Level (mbar)", NodeID: 20, DataType: "Double", Unit: "mbar", Description: "Vacuum level"},
		machine.TagDefinition{Name: "cooling_water_temp_c", DisplayName: "Cooling Water Temp (\u00b0C)", NodeID: 21, DataType: "Double", Unit: "°C", Description: "Cooling water temperature"},
		machine.TagDefinition{Name: "line_speed_m_min", DisplayName: "Line Speed (m/min)", NodeID: 22, DataType: "Double", Unit: "m/min", Description: "Line speed"},
	)
	return defs
}
