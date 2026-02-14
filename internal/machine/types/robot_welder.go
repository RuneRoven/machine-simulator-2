package types

import (
	"context"
	"time"

	"machine-simulator-2/internal/config"
	"machine-simulator-2/internal/machine"
)

func init() {
	machine.Register("robot_welder", func(cfg config.MachineInstanceConfig) machine.Machine {
		return NewRobotWelder(cfg)
	})
}

// RobotWelderMachine simulates a 6-axis robotic arc welding cell.
type RobotWelderMachine struct {
	*machine.BaseMachine

	joint1TempC         float64
	joint2TempC         float64
	joint3TempC         float64
	joint4TempC         float64
	joint5TempC         float64
	joint6TempC         float64
	arcVoltageV         float64
	arcCurrentA         float64
	wireFeedMMin        float64
	gasFlowLMin         float64
	travelSpeedMMS      float64
	weldPenetrationMM   float64
	torchAngleDeg       float64
	seamTrackingOffMM   float64
	spatterCount        float64
}

func NewRobotWelder(cfg config.MachineInstanceConfig) *RobotWelderMachine {
	return &RobotWelderMachine{
		BaseMachine:       machine.NewBaseMachine(cfg),
		joint1TempC:       42.0,
		joint2TempC:       44.0,
		joint3TempC:       40.0,
		joint4TempC:       38.0,
		joint5TempC:       39.0,
		joint6TempC:       37.0,
		arcVoltageV:       24.0,
		arcCurrentA:       220.0,
		wireFeedMMin:      8.5,
		gasFlowLMin:       15.0,
		travelSpeedMMS:    12.0,
		weldPenetrationMM: 3.2,
		torchAngleDeg:     15.0,
		seamTrackingOffMM: 0.1,
		spatterCount:      0.0,
	}
}

func (m *RobotWelderMachine) Start(ctx context.Context) {
	go m.updateTags(ctx)
	m.BaseMachine.Start(ctx)
}

func (m *RobotWelderMachine) updateTags(ctx context.Context) {
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
				for _, jt := range []*float64{&m.joint1TempC, &m.joint2TempC, &m.joint3TempC, &m.joint4TempC, &m.joint5TempC, &m.joint6TempC} {
					*jt += (rng.Float64()*2 - 1) * 0.3
					clampF(jt, 25.0, 70.0)
				}
				m.arcVoltageV += (rng.Float64()*2 - 1) * 0.3
				clampF(&m.arcVoltageV, 18.0, 32.0)
				m.arcCurrentA += (rng.Float64()*2 - 1) * 3.0
				clampF(&m.arcCurrentA, 150.0, 350.0)
				m.wireFeedMMin += (rng.Float64()*2 - 1) * 0.2
				clampF(&m.wireFeedMMin, 4.0, 15.0)
				m.gasFlowLMin += (rng.Float64()*2 - 1) * 0.3
				clampF(&m.gasFlowLMin, 10.0, 25.0)
				m.travelSpeedMMS += (rng.Float64()*2 - 1) * 0.5
				clampF(&m.travelSpeedMMS, 5.0, 25.0)
				m.weldPenetrationMM += (rng.Float64()*2 - 1) * 0.1
				clampF(&m.weldPenetrationMM, 1.5, 6.0)
				m.torchAngleDeg += (rng.Float64()*2 - 1) * 0.5
				clampF(&m.torchAngleDeg, 5.0, 45.0)
				m.seamTrackingOffMM += (rng.Float64()*2 - 1) * 0.05
				clampF(&m.seamTrackingOffMM, -1.0, 1.0)
				if rng.Float64() < 0.02 {
					m.spatterCount++
				}
			}
			m.Unlock()
		}
	}
}

func (m *RobotWelderMachine) GetTags() map[string]interface{} {
	m.RLock()
	defer m.RUnlock()
	tags := machine.BaseTagValues(m.BaseMachine)
	tags["joint1_temp_c"] = m.joint1TempC
	tags["joint2_temp_c"] = m.joint2TempC
	tags["joint3_temp_c"] = m.joint3TempC
	tags["joint4_temp_c"] = m.joint4TempC
	tags["joint5_temp_c"] = m.joint5TempC
	tags["joint6_temp_c"] = m.joint6TempC
	tags["arc_voltage_v"] = m.arcVoltageV
	tags["arc_current_a"] = m.arcCurrentA
	tags["wire_feed_m_min"] = m.wireFeedMMin
	tags["gas_flow_l_min"] = m.gasFlowLMin
	tags["travel_speed_mm_s"] = m.travelSpeedMMS
	tags["weld_penetration_mm"] = m.weldPenetrationMM
	tags["torch_angle_deg"] = m.torchAngleDeg
	tags["seam_tracking_offset_mm"] = m.seamTrackingOffMM
	tags["spatter_count"] = m.spatterCount
	return tags
}

func (m *RobotWelderMachine) GetTagDefinitions() []machine.TagDefinition {
	defs := machine.BaseTags()
	defs = append(defs,
		machine.TagDefinition{Name: "joint1_temp_c", DisplayName: "Joint 1 Temp (\u00b0C)", NodeID: 7, DataType: "Double", Unit: "°C", Description: "Joint 1 temperature"},
		machine.TagDefinition{Name: "joint2_temp_c", DisplayName: "Joint 2 Temp (\u00b0C)", NodeID: 8, DataType: "Double", Unit: "°C", Description: "Joint 2 temperature"},
		machine.TagDefinition{Name: "joint3_temp_c", DisplayName: "Joint 3 Temp (\u00b0C)", NodeID: 9, DataType: "Double", Unit: "°C", Description: "Joint 3 temperature"},
		machine.TagDefinition{Name: "joint4_temp_c", DisplayName: "Joint 4 Temp (\u00b0C)", NodeID: 10, DataType: "Double", Unit: "°C", Description: "Joint 4 temperature"},
		machine.TagDefinition{Name: "joint5_temp_c", DisplayName: "Joint 5 Temp (\u00b0C)", NodeID: 11, DataType: "Double", Unit: "°C", Description: "Joint 5 temperature"},
		machine.TagDefinition{Name: "joint6_temp_c", DisplayName: "Joint 6 Temp (\u00b0C)", NodeID: 12, DataType: "Double", Unit: "°C", Description: "Joint 6 temperature"},
		machine.TagDefinition{Name: "arc_voltage_v", DisplayName: "Arc Voltage (V)", NodeID: 13, DataType: "Double", Unit: "V", Description: "Arc voltage"},
		machine.TagDefinition{Name: "arc_current_a", DisplayName: "Arc Current (A)", NodeID: 14, DataType: "Double", Unit: "A", Description: "Arc current"},
		machine.TagDefinition{Name: "wire_feed_m_min", DisplayName: "Wire Feed Speed (m/min)", NodeID: 15, DataType: "Double", Unit: "m/min", Description: "Wire feed speed"},
		machine.TagDefinition{Name: "gas_flow_l_min", DisplayName: "Gas Flow Rate (L/min)", NodeID: 16, DataType: "Double", Unit: "L/min", Description: "Shielding gas flow rate"},
		machine.TagDefinition{Name: "travel_speed_mm_s", DisplayName: "Travel Speed (mm/s)", NodeID: 17, DataType: "Double", Unit: "mm/s", Description: "Torch travel speed"},
		machine.TagDefinition{Name: "weld_penetration_mm", DisplayName: "Weld Penetration (mm)", NodeID: 18, DataType: "Double", Unit: "mm", Description: "Weld penetration depth"},
		machine.TagDefinition{Name: "torch_angle_deg", DisplayName: "Torch Angle (\u00b0)", NodeID: 19, DataType: "Double", Unit: "°", Description: "Torch angle"},
		machine.TagDefinition{Name: "seam_tracking_offset_mm", DisplayName: "Seam Tracking Offset (mm)", NodeID: 20, DataType: "Double", Unit: "mm", Description: "Seam tracking offset"},
		machine.TagDefinition{Name: "spatter_count", DisplayName: "Spatter Count", NodeID: 21, DataType: "Double", Unit: "", Description: "Weld spatter event count"},
	)
	return defs
}
