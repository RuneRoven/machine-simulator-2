package types

import (
	"context"
	"time"

	"machine-simulator-2/internal/config"
	"machine-simulator-2/internal/machine"
)

func init() {
	machine.Register("corner_welder", func(cfg config.MachineInstanceConfig) machine.Machine {
		return NewCornerWelder(cfg)
	})
}

// CornerWelderMachine simulates a PVC window corner welding machine.
type CornerWelderMachine struct {
	*machine.BaseMachine

	weldTempC          float64
	weldTimeS          float64
	clampForceKN       float64
	jointStrengthN     float64
	alignmentAccMM     float64
	heatingPlateTempC  float64
	meltDepthMM        float64
	joiningPressBar    float64
	coolingTimeS       float64
	beadHeightMM       float64
	weldFlashMM        float64
	cornerAngleDeg     float64
	servoPositionMM    float64
	powerConsumptionKW float64
	cyclePressureBar   float64
}

func NewCornerWelder(cfg config.MachineInstanceConfig) *CornerWelderMachine {
	return &CornerWelderMachine{
		BaseMachine:        machine.NewBaseMachine(cfg),
		weldTempC:          260.0,
		weldTimeS:          25.0,
		clampForceKN:       8.0,
		jointStrengthN:     3500.0,
		alignmentAccMM:     0.15,
		heatingPlateTempC:  265.0,
		meltDepthMM:        2.5,
		joiningPressBar:    4.0,
		coolingTimeS:       15.0,
		beadHeightMM:       1.5,
		weldFlashMM:        0.3,
		cornerAngleDeg:     90.0,
		servoPositionMM:    0.0,
		powerConsumptionKW: 4.0,
		cyclePressureBar:   5.5,
	}
}

func (m *CornerWelderMachine) Start(ctx context.Context) {
	go m.updateTags(ctx)
	m.BaseMachine.Start(ctx)
}

func (m *CornerWelderMachine) updateTags(ctx context.Context) {
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
				m.weldTempC += (rng.Float64()*2 - 1) * 0.5
				clampF(&m.weldTempC, 240.0, 280.0)
				m.weldTimeS += (rng.Float64()*2 - 1) * 0.2
				clampF(&m.weldTimeS, 15.0, 40.0)
				m.clampForceKN += (rng.Float64()*2 - 1) * 0.1
				clampF(&m.clampForceKN, 4.0, 15.0)
				m.jointStrengthN += (rng.Float64()*2 - 1) * 20.0
				clampF(&m.jointStrengthN, 2500.0, 5000.0)
				m.alignmentAccMM += (rng.Float64()*2 - 1) * 0.005
				clampF(&m.alignmentAccMM, 0.05, 0.5)
				m.heatingPlateTempC += (rng.Float64()*2 - 1) * 0.5
				clampF(&m.heatingPlateTempC, 250.0, 285.0)
				m.meltDepthMM += (rng.Float64()*2 - 1) * 0.05
				clampF(&m.meltDepthMM, 1.5, 4.0)
				m.joiningPressBar += (rng.Float64()*2 - 1) * 0.1
				clampF(&m.joiningPressBar, 2.0, 7.0)
				m.coolingTimeS += (rng.Float64()*2 - 1) * 0.2
				clampF(&m.coolingTimeS, 8.0, 30.0)
				m.beadHeightMM += (rng.Float64()*2 - 1) * 0.02
				clampF(&m.beadHeightMM, 0.5, 3.0)
				m.weldFlashMM += (rng.Float64()*2 - 1) * 0.02
				clampF(&m.weldFlashMM, 0.0, 1.0)
				m.cornerAngleDeg += (rng.Float64()*2 - 1) * 0.01
				clampF(&m.cornerAngleDeg, 89.5, 90.5)
				m.servoPositionMM += (rng.Float64()*2 - 1) * 0.5
				clampF(&m.servoPositionMM, -50.0, 50.0)
				m.powerConsumptionKW += (rng.Float64()*2 - 1) * 0.1
				clampF(&m.powerConsumptionKW, 2.0, 8.0)
				m.cyclePressureBar += (rng.Float64()*2 - 1) * 0.1
				clampF(&m.cyclePressureBar, 3.0, 8.0)
			}
			m.Unlock()
		}
	}
}

func (m *CornerWelderMachine) GetTags() map[string]interface{} {
	m.RLock()
	defer m.RUnlock()
	tags := machine.BaseTagValues(m.BaseMachine)
	tags["weld_temp_c"] = m.weldTempC
	tags["weld_time_s"] = m.weldTimeS
	tags["clamp_force_kn"] = m.clampForceKN
	tags["joint_strength_n"] = m.jointStrengthN
	tags["alignment_accuracy_mm"] = m.alignmentAccMM
	tags["heating_plate_temp_c"] = m.heatingPlateTempC
	tags["melt_depth_mm"] = m.meltDepthMM
	tags["joining_pressure_bar"] = m.joiningPressBar
	tags["cooling_time_s"] = m.coolingTimeS
	tags["bead_height_mm"] = m.beadHeightMM
	tags["weld_flash_mm"] = m.weldFlashMM
	tags["corner_angle_deg"] = m.cornerAngleDeg
	tags["servo_position_mm"] = m.servoPositionMM
	tags["power_consumption_kw"] = m.powerConsumptionKW
	tags["cycle_pressure_bar"] = m.cyclePressureBar
	return tags
}

func (m *CornerWelderMachine) GetTagDefinitions() []machine.TagDefinition {
	defs := machine.BaseTags()
	defs = append(defs,
		machine.TagDefinition{Name: "weld_temp_c", DisplayName: "Weld Temp (°C)", NodeID: 7, DataType: "Double", Unit: "°C", Description: "Weld temperature"},
		machine.TagDefinition{Name: "weld_time_s", DisplayName: "Weld Time (s)", NodeID: 8, DataType: "Double", Unit: "s", Description: "Weld duration"},
		machine.TagDefinition{Name: "clamp_force_kn", DisplayName: "Clamp Force (kN)", NodeID: 9, DataType: "Double", Unit: "kN", Description: "Clamp force"},
		machine.TagDefinition{Name: "joint_strength_n", DisplayName: "Joint Strength (N)", NodeID: 10, DataType: "Double", Unit: "N", Description: "Weld joint strength"},
		machine.TagDefinition{Name: "alignment_accuracy_mm", DisplayName: "Alignment Accuracy (mm)", NodeID: 11, DataType: "Double", Unit: "mm", Description: "Profile alignment accuracy"},
		machine.TagDefinition{Name: "heating_plate_temp_c", DisplayName: "Heating Plate Temp (°C)", NodeID: 12, DataType: "Double", Unit: "°C", Description: "Heating plate temperature"},
		machine.TagDefinition{Name: "melt_depth_mm", DisplayName: "Melt Depth (mm)", NodeID: 13, DataType: "Double", Unit: "mm", Description: "Melt depth"},
		machine.TagDefinition{Name: "joining_pressure_bar", DisplayName: "Joining Pressure (bar)", NodeID: 14, DataType: "Double", Unit: "bar", Description: "Joining pressure"},
		machine.TagDefinition{Name: "cooling_time_s", DisplayName: "Cooling Time (s)", NodeID: 15, DataType: "Double", Unit: "s", Description: "Cooling time"},
		machine.TagDefinition{Name: "bead_height_mm", DisplayName: "Bead Height (mm)", NodeID: 16, DataType: "Double", Unit: "mm", Description: "Weld bead height"},
		machine.TagDefinition{Name: "weld_flash_mm", DisplayName: "Weld Flash (mm)", NodeID: 17, DataType: "Double", Unit: "mm", Description: "Weld flash height"},
		machine.TagDefinition{Name: "corner_angle_deg", DisplayName: "Corner Angle (°)", NodeID: 18, DataType: "Double", Unit: "°", Description: "Corner angle"},
		machine.TagDefinition{Name: "servo_position_mm", DisplayName: "Servo Position (mm)", NodeID: 19, DataType: "Double", Unit: "mm", Description: "Servo position"},
		machine.TagDefinition{Name: "power_consumption_kw", DisplayName: "Power Consumption (kW)", NodeID: 20, DataType: "Double", Unit: "kW", Description: "Power consumption"},
		machine.TagDefinition{Name: "cycle_pressure_bar", DisplayName: "Cycle Pressure (bar)", NodeID: 21, DataType: "Double", Unit: "bar", Description: "Cycle pressure"},
	)
	return defs
}
