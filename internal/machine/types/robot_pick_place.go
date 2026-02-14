package types

import (
	"context"
	"time"

	"machine-simulator-2/internal/config"
	"machine-simulator-2/internal/machine"
)

func init() {
	machine.Register("robot_pick_place", func(cfg config.MachineInstanceConfig) machine.Machine {
		return NewRobotPickPlace(cfg)
	})
}

// RobotPickPlaceMachine simulates a 6-axis pick-and-place robot.
type RobotPickPlaceMachine struct {
	*machine.BaseMachine

	joint1Angle    float64
	joint2Angle    float64
	joint3Angle    float64
	joint4Angle    float64
	joint5Angle    float64
	joint6Angle    float64
	tcpX           float64
	tcpY           float64
	tcpZ           float64
	gripperForceN  float64
	vacuumPressKPa float64
	pickAccuracyUM float64
	placeAccuracyUM float64
	cycleSpeedPct  float64
	payloadKg      float64
	accelMS2       float64
}

func NewRobotPickPlace(cfg config.MachineInstanceConfig) *RobotPickPlaceMachine {
	return &RobotPickPlaceMachine{
		BaseMachine:     machine.NewBaseMachine(cfg),
		joint1Angle:     0.0,
		joint2Angle:     -30.0,
		joint3Angle:     60.0,
		joint4Angle:     0.0,
		joint5Angle:     -45.0,
		joint6Angle:     0.0,
		tcpX:            500.0,
		tcpY:            0.0,
		tcpZ:            300.0,
		gripperForceN:   50.0,
		vacuumPressKPa:  -60.0,
		pickAccuracyUM:  25.0,
		placeAccuracyUM: 30.0,
		cycleSpeedPct:   85.0,
		payloadKg:       2.5,
		accelMS2:        8.0,
	}
}

func (m *RobotPickPlaceMachine) Start(ctx context.Context) {
	go m.updateTags(ctx)
	m.BaseMachine.Start(ctx)
}

func (m *RobotPickPlaceMachine) updateTags(ctx context.Context) {
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
				m.joint1Angle += (rng.Float64()*2 - 1) * 2.0
				clampF(&m.joint1Angle, -180.0, 180.0)
				m.joint2Angle += (rng.Float64()*2 - 1) * 1.5
				clampF(&m.joint2Angle, -90.0, 90.0)
				m.joint3Angle += (rng.Float64()*2 - 1) * 2.0
				clampF(&m.joint3Angle, -120.0, 120.0)
				m.joint4Angle += (rng.Float64()*2 - 1) * 2.0
				clampF(&m.joint4Angle, -180.0, 180.0)
				m.joint5Angle += (rng.Float64()*2 - 1) * 1.5
				clampF(&m.joint5Angle, -120.0, 120.0)
				m.joint6Angle += (rng.Float64()*2 - 1) * 2.0
				clampF(&m.joint6Angle, -360.0, 360.0)
				m.tcpX += (rng.Float64()*2 - 1) * 5.0
				clampF(&m.tcpX, -1000.0, 1000.0)
				m.tcpY += (rng.Float64()*2 - 1) * 5.0
				clampF(&m.tcpY, -1000.0, 1000.0)
				m.tcpZ += (rng.Float64()*2 - 1) * 3.0
				clampF(&m.tcpZ, 0.0, 800.0)
				m.gripperForceN += (rng.Float64()*2 - 1) * 1.0
				clampF(&m.gripperForceN, 10.0, 150.0)
				m.vacuumPressKPa += (rng.Float64()*2 - 1) * 1.0
				clampF(&m.vacuumPressKPa, -85.0, -20.0)
				m.pickAccuracyUM += (rng.Float64()*2 - 1) * 1.0
				clampF(&m.pickAccuracyUM, 5.0, 100.0)
				m.placeAccuracyUM += (rng.Float64()*2 - 1) * 1.0
				clampF(&m.placeAccuracyUM, 5.0, 100.0)
				m.cycleSpeedPct += (rng.Float64()*2 - 1) * 0.5
				clampF(&m.cycleSpeedPct, 50.0, 100.0)
				m.payloadKg += (rng.Float64()*2 - 1) * 0.1
				clampF(&m.payloadKg, 0.5, 10.0)
				m.accelMS2 += (rng.Float64()*2 - 1) * 0.2
				clampF(&m.accelMS2, 3.0, 15.0)
			}
			m.Unlock()
		}
	}
}

func (m *RobotPickPlaceMachine) GetTags() map[string]interface{} {
	m.RLock()
	defer m.RUnlock()
	tags := machine.BaseTagValues(m.BaseMachine)
	tags["joint1_angle"] = m.joint1Angle
	tags["joint2_angle"] = m.joint2Angle
	tags["joint3_angle"] = m.joint3Angle
	tags["joint4_angle"] = m.joint4Angle
	tags["joint5_angle"] = m.joint5Angle
	tags["joint6_angle"] = m.joint6Angle
	tags["tcp_x"] = m.tcpX
	tags["tcp_y"] = m.tcpY
	tags["tcp_z"] = m.tcpZ
	tags["gripper_force_n"] = m.gripperForceN
	tags["vacuum_pressure_kpa"] = m.vacuumPressKPa
	tags["pick_accuracy_um"] = m.pickAccuracyUM
	tags["place_accuracy_um"] = m.placeAccuracyUM
	tags["cycle_speed_pct"] = m.cycleSpeedPct
	tags["payload_kg"] = m.payloadKg
	tags["acceleration_m_s2"] = m.accelMS2
	return tags
}

func (m *RobotPickPlaceMachine) GetTagDefinitions() []machine.TagDefinition {
	defs := machine.BaseTags()
	defs = append(defs,
		machine.TagDefinition{Name: "joint1_angle", DisplayName: "Joint 1 Angle (\u00b0)", NodeID: 7, DataType: "Double", Unit: "°", Description: "Joint 1 angle"},
		machine.TagDefinition{Name: "joint2_angle", DisplayName: "Joint 2 Angle (\u00b0)", NodeID: 8, DataType: "Double", Unit: "°", Description: "Joint 2 angle"},
		machine.TagDefinition{Name: "joint3_angle", DisplayName: "Joint 3 Angle (\u00b0)", NodeID: 9, DataType: "Double", Unit: "°", Description: "Joint 3 angle"},
		machine.TagDefinition{Name: "joint4_angle", DisplayName: "Joint 4 Angle (\u00b0)", NodeID: 10, DataType: "Double", Unit: "°", Description: "Joint 4 angle"},
		machine.TagDefinition{Name: "joint5_angle", DisplayName: "Joint 5 Angle (\u00b0)", NodeID: 11, DataType: "Double", Unit: "°", Description: "Joint 5 angle"},
		machine.TagDefinition{Name: "joint6_angle", DisplayName: "Joint 6 Angle (\u00b0)", NodeID: 12, DataType: "Double", Unit: "°", Description: "Joint 6 angle"},
		machine.TagDefinition{Name: "tcp_x", DisplayName: "TCP X Position (mm)", NodeID: 13, DataType: "Double", Unit: "mm", Description: "Tool center point X position"},
		machine.TagDefinition{Name: "tcp_y", DisplayName: "TCP Y Position (mm)", NodeID: 14, DataType: "Double", Unit: "mm", Description: "Tool center point Y position"},
		machine.TagDefinition{Name: "tcp_z", DisplayName: "TCP Z Position (mm)", NodeID: 15, DataType: "Double", Unit: "mm", Description: "Tool center point Z position"},
		machine.TagDefinition{Name: "gripper_force_n", DisplayName: "Gripper Force (N)", NodeID: 16, DataType: "Double", Unit: "N", Description: "Gripper force"},
		machine.TagDefinition{Name: "vacuum_pressure_kpa", DisplayName: "Vacuum Pressure (kPa)", NodeID: 17, DataType: "Double", Unit: "kPa", Description: "Vacuum pressure for suction gripper"},
		machine.TagDefinition{Name: "pick_accuracy_um", DisplayName: "Pick Accuracy (\u00b5m)", NodeID: 18, DataType: "Double", Unit: "µm", Description: "Pick position accuracy"},
		machine.TagDefinition{Name: "place_accuracy_um", DisplayName: "Place Accuracy (\u00b5m)", NodeID: 19, DataType: "Double", Unit: "µm", Description: "Place position accuracy"},
		machine.TagDefinition{Name: "cycle_speed_pct", DisplayName: "Cycle Speed (%)", NodeID: 20, DataType: "Double", Unit: "%", Description: "Cycle speed percentage"},
		machine.TagDefinition{Name: "payload_kg", DisplayName: "Payload Weight (kg)", NodeID: 21, DataType: "Double", Unit: "kg", Description: "Current payload weight"},
		machine.TagDefinition{Name: "acceleration_m_s2", DisplayName: "Acceleration (m/s\u00b2)", NodeID: 22, DataType: "Double", Unit: "m/s²", Description: "Acceleration"},
	)
	return defs
}
