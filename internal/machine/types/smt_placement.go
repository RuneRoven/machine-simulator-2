package types

import (
	"context"
	"time"

	"machine-simulator-2/internal/config"
	"machine-simulator-2/internal/machine"
)

func init() {
	machine.Register("smt_placement", func(cfg config.MachineInstanceConfig) machine.Machine {
		return NewSMTPlacement(cfg)
	})
}

// SMTPlacementMachine simulates a surface-mount technology pick-and-place machine.
type SMTPlacementMachine struct {
	*machine.BaseMachine

	placementRateCPH  float64
	nozzleVacuumKPa   float64
	feederStatusPct   float64
	placementAccUM    float64
	headXMM           float64
	headYMM           float64
	visionScorePct    float64
	componentCount    float64
	rejectCount       float64
	nozzleTempC       float64
	boardTempC        float64
	tapeTensionN      float64
	conveyorWidthMM   float64
	rotationAccDeg    float64
	pickFailureCount  float64
}

func NewSMTPlacement(cfg config.MachineInstanceConfig) *SMTPlacementMachine {
	return &SMTPlacementMachine{
		BaseMachine:      machine.NewBaseMachine(cfg),
		placementRateCPH: 25000.0,
		nozzleVacuumKPa:  -55.0,
		feederStatusPct:  98.0,
		placementAccUM:   35.0,
		headXMM:          200.0,
		headYMM:          150.0,
		visionScorePct:   96.0,
		componentCount:   0.0,
		rejectCount:      0.0,
		nozzleTempC:      28.0,
		boardTempC:       25.0,
		tapeTensionN:     2.5,
		conveyorWidthMM:  250.0,
		rotationAccDeg:   0.02,
		pickFailureCount: 0.0,
	}
}

func (m *SMTPlacementMachine) Start(ctx context.Context) {
	go m.updateTags(ctx)
	m.BaseMachine.Start(ctx)
}

func (m *SMTPlacementMachine) updateTags(ctx context.Context) {
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
				m.placementRateCPH += (rng.Float64()*2 - 1) * 100.0
				clampF(&m.placementRateCPH, 15000.0, 40000.0)
				m.nozzleVacuumKPa += (rng.Float64()*2 - 1) * 0.5
				clampF(&m.nozzleVacuumKPa, -80.0, -30.0)
				m.feederStatusPct += (rng.Float64()*2 - 1) * 0.2
				clampF(&m.feederStatusPct, 80.0, 100.0)
				m.placementAccUM += (rng.Float64()*2 - 1) * 1.0
				clampF(&m.placementAccUM, 10.0, 80.0)
				m.headXMM += (rng.Float64()*2 - 1) * 10.0
				clampF(&m.headXMM, 0.0, 500.0)
				m.headYMM += (rng.Float64()*2 - 1) * 10.0
				clampF(&m.headYMM, 0.0, 400.0)
				m.visionScorePct += (rng.Float64()*2 - 1) * 0.3
				clampF(&m.visionScorePct, 80.0, 100.0)
				m.componentCount += rng.Float64() * 5.0
				m.nozzleTempC += (rng.Float64()*2 - 1) * 0.2
				clampF(&m.nozzleTempC, 20.0, 45.0)
				m.boardTempC += (rng.Float64()*2 - 1) * 0.1
				clampF(&m.boardTempC, 20.0, 40.0)
				m.tapeTensionN += (rng.Float64()*2 - 1) * 0.1
				clampF(&m.tapeTensionN, 1.0, 5.0)
				m.conveyorWidthMM += (rng.Float64()*2 - 1) * 0.1
				clampF(&m.conveyorWidthMM, 200.0, 350.0)
				m.rotationAccDeg += (rng.Float64()*2 - 1) * 0.002
				clampF(&m.rotationAccDeg, 0.005, 0.1)
				if rng.Float64() < 0.005 {
					m.rejectCount++
				}
				if rng.Float64() < 0.003 {
					m.pickFailureCount++
				}
			}
			m.Unlock()
		}
	}
}

func (m *SMTPlacementMachine) GetTags() map[string]interface{} {
	m.RLock()
	defer m.RUnlock()
	tags := machine.BaseTagValues(m.BaseMachine)
	tags["placement_rate_cph"] = m.placementRateCPH
	tags["nozzle_vacuum_kpa"] = m.nozzleVacuumKPa
	tags["feeder_status_pct"] = m.feederStatusPct
	tags["placement_accuracy_um"] = m.placementAccUM
	tags["head_x_mm"] = m.headXMM
	tags["head_y_mm"] = m.headYMM
	tags["vision_score_pct"] = m.visionScorePct
	tags["component_count"] = m.componentCount
	tags["reject_count"] = m.rejectCount
	tags["nozzle_temp_c"] = m.nozzleTempC
	tags["board_temp_c"] = m.boardTempC
	tags["tape_tension_n"] = m.tapeTensionN
	tags["conveyor_width_mm"] = m.conveyorWidthMM
	tags["rotation_accuracy_deg"] = m.rotationAccDeg
	tags["pick_failure_count"] = m.pickFailureCount
	return tags
}

func (m *SMTPlacementMachine) GetTagDefinitions() []machine.TagDefinition {
	defs := machine.BaseTags()
	defs = append(defs,
		machine.TagDefinition{Name: "placement_rate_cph", DisplayName: "Placement Rate (cph)", NodeID: 7, DataType: "Double", Unit: "cph", Description: "Component placement rate"},
		machine.TagDefinition{Name: "nozzle_vacuum_kpa", DisplayName: "Nozzle Vacuum (kPa)", NodeID: 8, DataType: "Double", Unit: "kPa", Description: "Nozzle vacuum pressure"},
		machine.TagDefinition{Name: "feeder_status_pct", DisplayName: "Feeder Availability (%)", NodeID: 9, DataType: "Double", Unit: "%", Description: "Feeder availability"},
		machine.TagDefinition{Name: "placement_accuracy_um", DisplayName: "Placement Accuracy (\u00b5m)", NodeID: 10, DataType: "Double", Unit: "µm", Description: "Placement accuracy"},
		machine.TagDefinition{Name: "head_x_mm", DisplayName: "Head X Position (mm)", NodeID: 11, DataType: "Double", Unit: "mm", Description: "Head X position"},
		machine.TagDefinition{Name: "head_y_mm", DisplayName: "Head Y Position (mm)", NodeID: 12, DataType: "Double", Unit: "mm", Description: "Head Y position"},
		machine.TagDefinition{Name: "vision_score_pct", DisplayName: "Vision Score (%)", NodeID: 13, DataType: "Double", Unit: "%", Description: "Vision system alignment score"},
		machine.TagDefinition{Name: "component_count", DisplayName: "Component Count", NodeID: 14, DataType: "Double", Unit: "", Description: "Components placed"},
		machine.TagDefinition{Name: "reject_count", DisplayName: "Reject Count", NodeID: 15, DataType: "Double", Unit: "", Description: "Rejected placements"},
		machine.TagDefinition{Name: "nozzle_temp_c", DisplayName: "Nozzle Temp (\u00b0C)", NodeID: 16, DataType: "Double", Unit: "°C", Description: "Nozzle temperature"},
		machine.TagDefinition{Name: "board_temp_c", DisplayName: "Board Temp (\u00b0C)", NodeID: 17, DataType: "Double", Unit: "°C", Description: "Board temperature"},
		machine.TagDefinition{Name: "tape_tension_n", DisplayName: "Tape Tension (N)", NodeID: 18, DataType: "Double", Unit: "N", Description: "Tape feeder tension"},
		machine.TagDefinition{Name: "conveyor_width_mm", DisplayName: "Conveyor Width (mm)", NodeID: 19, DataType: "Double", Unit: "mm", Description: "Conveyor width setting"},
		machine.TagDefinition{Name: "rotation_accuracy_deg", DisplayName: "Rotation Accuracy (\u00b0)", NodeID: 20, DataType: "Double", Unit: "°", Description: "Component rotation accuracy"},
		machine.TagDefinition{Name: "pick_failure_count", DisplayName: "Pick Failure Count", NodeID: 21, DataType: "Double", Unit: "", Description: "Pick failure count"},
	)
	return defs
}
