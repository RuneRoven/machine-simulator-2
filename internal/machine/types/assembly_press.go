package types

import (
	"context"
	"time"

	"machine-simulator-2/internal/config"
	"machine-simulator-2/internal/machine"
)

func init() {
	machine.Register("assembly_press", func(cfg config.MachineInstanceConfig) machine.Machine {
		return NewAssemblyPress(cfg)
	})
}

// AssemblyPressMachine simulates a furniture assembly press (dowel/cam insertion).
type AssemblyPressMachine struct {
	*machine.BaseMachine

	pressForceKN     float64
	alignmentAccMM   float64
	dwellTimeMS      float64
	cyclePressureBar float64
	ramPositionMM    float64
	insertionDepthMM float64
	parallelismUM    float64
	strokeSpeedMMS   float64
	partPresencePct  float64
	forceCurvePeakKN float64
	workEnergyJ      float64
	temperatureC     float64
	cushionPressBar  float64
	ejectorForceN    float64
	servoCurrentA    float64
}

func NewAssemblyPress(cfg config.MachineInstanceConfig) *AssemblyPressMachine {
	return &AssemblyPressMachine{
		BaseMachine:      machine.NewBaseMachine(cfg),
		pressForceKN:     15.0,
		alignmentAccMM:   0.1,
		dwellTimeMS:      500.0,
		cyclePressureBar: 6.0,
		ramPositionMM:    0.0,
		insertionDepthMM: 30.0,
		parallelismUM:    20.0,
		strokeSpeedMMS:   80.0,
		partPresencePct:  100.0,
		forceCurvePeakKN: 18.0,
		workEnergyJ:      45.0,
		temperatureC:     28.0,
		cushionPressBar:  2.0,
		ejectorForceN:    200.0,
		servoCurrentA:    6.0,
	}
}

func (m *AssemblyPressMachine) Start(ctx context.Context) {
	go m.updateTags(ctx)
	m.BaseMachine.Start(ctx)
}

func (m *AssemblyPressMachine) updateTags(ctx context.Context) {
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
				m.pressForceKN += (rng.Float64()*2 - 1) * 0.3
				clampF(&m.pressForceKN, 5.0, 50.0)
				m.alignmentAccMM += (rng.Float64()*2 - 1) * 0.005
				clampF(&m.alignmentAccMM, 0.02, 0.5)
				m.dwellTimeMS += (rng.Float64()*2 - 1) * 5.0
				clampF(&m.dwellTimeMS, 200.0, 2000.0)
				m.cyclePressureBar += (rng.Float64()*2 - 1) * 0.1
				clampF(&m.cyclePressureBar, 3.0, 10.0)
				m.ramPositionMM += (rng.Float64()*2 - 1) * 2.0
				clampF(&m.ramPositionMM, -50.0, 50.0)
				m.insertionDepthMM += (rng.Float64()*2 - 1) * 0.1
				clampF(&m.insertionDepthMM, 10.0, 60.0)
				m.parallelismUM += (rng.Float64()*2 - 1) * 1.0
				clampF(&m.parallelismUM, 5.0, 80.0)
				m.strokeSpeedMMS += (rng.Float64()*2 - 1) * 1.0
				clampF(&m.strokeSpeedMMS, 30.0, 150.0)
				m.partPresencePct += (rng.Float64()*2 - 1) * 0.1
				clampF(&m.partPresencePct, 90.0, 100.0)
				m.forceCurvePeakKN += (rng.Float64()*2 - 1) * 0.5
				clampF(&m.forceCurvePeakKN, 5.0, 55.0)
				m.workEnergyJ += (rng.Float64()*2 - 1) * 1.0
				clampF(&m.workEnergyJ, 10.0, 120.0)
				m.temperatureC += (rng.Float64()*2 - 1) * 0.2
				clampF(&m.temperatureC, 18.0, 45.0)
				m.cushionPressBar += (rng.Float64()*2 - 1) * 0.05
				clampF(&m.cushionPressBar, 0.5, 5.0)
				m.ejectorForceN += (rng.Float64()*2 - 1) * 3.0
				clampF(&m.ejectorForceN, 50.0, 500.0)
				m.servoCurrentA += (rng.Float64()*2 - 1) * 0.1
				clampF(&m.servoCurrentA, 2.0, 15.0)
			}
			m.Unlock()
		}
	}
}

func (m *AssemblyPressMachine) GetTags() map[string]interface{} {
	m.RLock()
	defer m.RUnlock()
	tags := machine.BaseTagValues(m.BaseMachine)
	tags["press_force_kn"] = m.pressForceKN
	tags["alignment_accuracy_mm"] = m.alignmentAccMM
	tags["dwell_time_ms"] = m.dwellTimeMS
	tags["cycle_pressure_bar"] = m.cyclePressureBar
	tags["ram_position_mm"] = m.ramPositionMM
	tags["insertion_depth_mm"] = m.insertionDepthMM
	tags["parallelism_um"] = m.parallelismUM
	tags["stroke_speed_mm_s"] = m.strokeSpeedMMS
	tags["part_presence_pct"] = m.partPresencePct
	tags["force_curve_peak_kn"] = m.forceCurvePeakKN
	tags["work_energy_j"] = m.workEnergyJ
	tags["temperature_c"] = m.temperatureC
	tags["cushion_pressure_bar"] = m.cushionPressBar
	tags["ejector_force_n"] = m.ejectorForceN
	tags["servo_current_a"] = m.servoCurrentA
	return tags
}

func (m *AssemblyPressMachine) GetTagDefinitions() []machine.TagDefinition {
	defs := machine.BaseTags()
	defs = append(defs,
		machine.TagDefinition{Name: "press_force_kn", DisplayName: "Press Force (kN)", NodeID: 7, DataType: "Double", Unit: "kN", Description: "Press force"},
		machine.TagDefinition{Name: "alignment_accuracy_mm", DisplayName: "Alignment Accuracy (mm)", NodeID: 8, DataType: "Double", Unit: "mm", Description: "Alignment accuracy"},
		machine.TagDefinition{Name: "dwell_time_ms", DisplayName: "Dwell Time (ms)", NodeID: 9, DataType: "Double", Unit: "ms", Description: "Press dwell time"},
		machine.TagDefinition{Name: "cycle_pressure_bar", DisplayName: "Cycle Pressure (bar)", NodeID: 10, DataType: "Double", Unit: "bar", Description: "Cycle pressure"},
		machine.TagDefinition{Name: "ram_position_mm", DisplayName: "Ram Position (mm)", NodeID: 11, DataType: "Double", Unit: "mm", Description: "Ram position"},
		machine.TagDefinition{Name: "insertion_depth_mm", DisplayName: "Insertion Depth (mm)", NodeID: 12, DataType: "Double", Unit: "mm", Description: "Insertion depth"},
		machine.TagDefinition{Name: "parallelism_um", DisplayName: "Platen Parallelism (\u00b5m)", NodeID: 13, DataType: "Double", Unit: "µm", Description: "Platen parallelism"},
		machine.TagDefinition{Name: "stroke_speed_mm_s", DisplayName: "Stroke Speed (mm/s)", NodeID: 14, DataType: "Double", Unit: "mm/s", Description: "Stroke speed"},
		machine.TagDefinition{Name: "part_presence_pct", DisplayName: "Part Presence (%)", NodeID: 15, DataType: "Double", Unit: "%", Description: "Part presence sensor"},
		machine.TagDefinition{Name: "force_curve_peak_kn", DisplayName: "Peak Force (kN)", NodeID: 16, DataType: "Double", Unit: "kN", Description: "Peak force in cycle"},
		machine.TagDefinition{Name: "work_energy_j", DisplayName: "Work Energy (J)", NodeID: 17, DataType: "Double", Unit: "J", Description: "Work energy per cycle"},
		machine.TagDefinition{Name: "temperature_c", DisplayName: "Temperature (\u00b0C)", NodeID: 18, DataType: "Double", Unit: "°C", Description: "Ambient temperature"},
		machine.TagDefinition{Name: "cushion_pressure_bar", DisplayName: "Cushion Pressure (bar)", NodeID: 19, DataType: "Double", Unit: "bar", Description: "Cushion pressure"},
		machine.TagDefinition{Name: "ejector_force_n", DisplayName: "Ejector Force (N)", NodeID: 20, DataType: "Double", Unit: "N", Description: "Ejector force"},
		machine.TagDefinition{Name: "servo_current_a", DisplayName: "Servo Current (A)", NodeID: 21, DataType: "Double", Unit: "A", Description: "Servo motor current"},
	)
	return defs
}
