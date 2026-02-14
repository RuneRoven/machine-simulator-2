package types

import (
	"context"
	"time"

	"machine-simulator-2/internal/config"
	"machine-simulator-2/internal/machine"
)

func init() {
	machine.Register("capping_machine", func(cfg config.MachineInstanceConfig) machine.Machine {
		return NewCappingMachine(cfg)
	})
}

// CappingMachineMachine simulates a pharmaceutical capping machine.
type CappingMachineMachine struct {
	*machine.BaseMachine

	torqueNM             float64
	capPosAccuracyMM     float64
	rejectRatePct        float64
	feedRateCapsMin      float64
	capPresencePct       float64
	sealIntegrityPct     float64
	chuckSpeedRPM        float64
	camTrackForceN       float64
	capHeightMM          float64
	skirtDiameterMM      float64
	linerCompressionPct  float64
	elevatorLevelPct     float64
	servoCurrentA        float64
	vibrationMMS         float64
	capOrientationPct    float64
}

func NewCappingMachine(cfg config.MachineInstanceConfig) *CappingMachineMachine {
	return &CappingMachineMachine{
		BaseMachine:         machine.NewBaseMachine(cfg),
		torqueNM:            1.8,
		capPosAccuracyMM:    0.2,
		rejectRatePct:       0.5,
		feedRateCapsMin:     120.0,
		capPresencePct:      99.5,
		sealIntegrityPct:    99.8,
		chuckSpeedRPM:       600.0,
		camTrackForceN:      150.0,
		capHeightMM:         20.0,
		skirtDiameterMM:     28.0,
		linerCompressionPct: 30.0,
		elevatorLevelPct:    75.0,
		servoCurrentA:       4.5,
		vibrationMMS:        1.5,
		capOrientationPct:   98.0,
	}
}

func (m *CappingMachineMachine) Start(ctx context.Context) {
	go m.updateTags(ctx)
	m.BaseMachine.Start(ctx)
}

func (m *CappingMachineMachine) updateTags(ctx context.Context) {
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
				m.torqueNM += (rng.Float64()*2 - 1) * 0.05
				clampF(&m.torqueNM, 0.5, 4.0)
				m.capPosAccuracyMM += (rng.Float64()*2 - 1) * 0.01
				clampF(&m.capPosAccuracyMM, 0.05, 0.5)
				m.rejectRatePct += (rng.Float64()*2 - 1) * 0.05
				clampF(&m.rejectRatePct, 0.0, 5.0)
				m.feedRateCapsMin += (rng.Float64()*2 - 1) * 1.0
				clampF(&m.feedRateCapsMin, 60.0, 300.0)
				m.capPresencePct += (rng.Float64()*2 - 1) * 0.05
				clampF(&m.capPresencePct, 95.0, 100.0)
				m.sealIntegrityPct += (rng.Float64()*2 - 1) * 0.02
				clampF(&m.sealIntegrityPct, 95.0, 100.0)
				m.chuckSpeedRPM += (rng.Float64()*2 - 1) * 5.0
				clampF(&m.chuckSpeedRPM, 200.0, 1200.0)
				m.camTrackForceN += (rng.Float64()*2 - 1) * 2.0
				clampF(&m.camTrackForceN, 50.0, 300.0)
				m.capHeightMM += (rng.Float64()*2 - 1) * 0.05
				clampF(&m.capHeightMM, 15.0, 30.0)
				m.skirtDiameterMM += (rng.Float64()*2 - 1) * 0.02
				clampF(&m.skirtDiameterMM, 20.0, 40.0)
				m.linerCompressionPct += (rng.Float64()*2 - 1) * 0.5
				clampF(&m.linerCompressionPct, 10.0, 60.0)
				m.elevatorLevelPct -= rng.Float64() * 0.02
				clampF(&m.elevatorLevelPct, 10.0, 100.0)
				m.servoCurrentA += (rng.Float64()*2 - 1) * 0.1
				clampF(&m.servoCurrentA, 1.0, 10.0)
				m.vibrationMMS += (rng.Float64()*2 - 1) * 0.1
				clampF(&m.vibrationMMS, 0.3, 5.0)
				m.capOrientationPct += (rng.Float64()*2 - 1) * 0.1
				clampF(&m.capOrientationPct, 90.0, 100.0)
			}
			m.Unlock()
		}
	}
}

func (m *CappingMachineMachine) GetTags() map[string]interface{} {
	m.RLock()
	defer m.RUnlock()
	tags := machine.BaseTagValues(m.BaseMachine)
	tags["torque_nm"] = m.torqueNM
	tags["cap_position_accuracy_mm"] = m.capPosAccuracyMM
	tags["reject_rate_pct"] = m.rejectRatePct
	tags["feed_rate_caps_min"] = m.feedRateCapsMin
	tags["cap_presence_pct"] = m.capPresencePct
	tags["seal_integrity_pct"] = m.sealIntegrityPct
	tags["chuck_speed_rpm"] = m.chuckSpeedRPM
	tags["cam_track_force_n"] = m.camTrackForceN
	tags["cap_height_mm"] = m.capHeightMM
	tags["skirt_diameter_mm"] = m.skirtDiameterMM
	tags["liner_compression_pct"] = m.linerCompressionPct
	tags["elevator_level_pct"] = m.elevatorLevelPct
	tags["servo_current_a"] = m.servoCurrentA
	tags["vibration_mm_s"] = m.vibrationMMS
	tags["cap_orientation_score_pct"] = m.capOrientationPct
	return tags
}

func (m *CappingMachineMachine) GetTagDefinitions() []machine.TagDefinition {
	defs := machine.BaseTags()
	defs = append(defs,
		machine.TagDefinition{Name: "torque_nm", DisplayName: "Torque (N·m)", NodeID: 7, DataType: "Double", Unit: "N·m", Description: "Capping torque"},
		machine.TagDefinition{Name: "cap_position_accuracy_mm", DisplayName: "Cap Position Accuracy (mm)", NodeID: 8, DataType: "Double", Unit: "mm", Description: "Cap position accuracy"},
		machine.TagDefinition{Name: "reject_rate_pct", DisplayName: "Reject Rate (%)", NodeID: 9, DataType: "Double", Unit: "%", Description: "Reject rate"},
		machine.TagDefinition{Name: "feed_rate_caps_min", DisplayName: "Feed Rate (caps/min)", NodeID: 10, DataType: "Double", Unit: "caps/min", Description: "Cap feed rate"},
		machine.TagDefinition{Name: "cap_presence_pct", DisplayName: "Cap Presence (%)", NodeID: 11, DataType: "Double", Unit: "%", Description: "Cap presence detection"},
		machine.TagDefinition{Name: "seal_integrity_pct", DisplayName: "Seal Integrity (%)", NodeID: 12, DataType: "Double", Unit: "%", Description: "Seal integrity"},
		machine.TagDefinition{Name: "chuck_speed_rpm", DisplayName: "Chuck Speed (RPM)", NodeID: 13, DataType: "Double", Unit: "RPM", Description: "Chuck rotation speed"},
		machine.TagDefinition{Name: "cam_track_force_n", DisplayName: "Cam Track Force (N)", NodeID: 14, DataType: "Double", Unit: "N", Description: "Cam track force"},
		machine.TagDefinition{Name: "cap_height_mm", DisplayName: "Cap Height (mm)", NodeID: 15, DataType: "Double", Unit: "mm", Description: "Cap height"},
		machine.TagDefinition{Name: "skirt_diameter_mm", DisplayName: "Skirt Diameter (mm)", NodeID: 16, DataType: "Double", Unit: "mm", Description: "Skirt diameter"},
		machine.TagDefinition{Name: "liner_compression_pct", DisplayName: "Liner Compression (%)", NodeID: 17, DataType: "Double", Unit: "%", Description: "Liner compression"},
		machine.TagDefinition{Name: "elevator_level_pct", DisplayName: "Elevator Level (%)", NodeID: 18, DataType: "Double", Unit: "%", Description: "Cap elevator level"},
		machine.TagDefinition{Name: "servo_current_a", DisplayName: "Servo Current (A)", NodeID: 19, DataType: "Double", Unit: "A", Description: "Servo motor current"},
		machine.TagDefinition{Name: "vibration_mm_s", DisplayName: "Vibration (mm/s)", NodeID: 20, DataType: "Double", Unit: "mm/s", Description: "Vibration level"},
		machine.TagDefinition{Name: "cap_orientation_score_pct", DisplayName: "Cap Orientation Score (%)", NodeID: 21, DataType: "Double", Unit: "%", Description: "Cap orientation score"},
	)
	return defs
}
