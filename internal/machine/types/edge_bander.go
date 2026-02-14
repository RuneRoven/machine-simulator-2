package types

import (
	"context"
	"time"

	"machine-simulator-2/internal/config"
	"machine-simulator-2/internal/machine"
)

func init() {
	machine.Register("edge_bander", func(cfg config.MachineInstanceConfig) machine.Machine {
		return NewEdgeBander(cfg)
	})
}

// EdgeBanderMachine simulates a furniture edge banding machine.
type EdgeBanderMachine struct {
	*machine.BaseMachine

	glueTempC           float64
	feedSpeedMMin       float64
	trimmerPositionMM   float64
	tapeTensionN        float64
	rollerForceN        float64
	edgeThicknessMM     float64
	glueFlowGMin        float64
	panelThicknessMM    float64
	trimQualityScorePct float64
	pressureZoneBar     float64
	preMillingDepthMM   float64
	hotAirTempC         float64
	jointStrengthNCM    float64
	scraperPositionMM   float64
	buffingSpeedRPM     float64
}

func NewEdgeBander(cfg config.MachineInstanceConfig) *EdgeBanderMachine {
	return &EdgeBanderMachine{
		BaseMachine:         machine.NewBaseMachine(cfg),
		glueTempC:           200.0,
		feedSpeedMMin:       12.0,
		trimmerPositionMM:   0.5,
		tapeTensionN:        20.0,
		rollerForceN:        250.0,
		edgeThicknessMM:     0.8,
		glueFlowGMin:        25.0,
		panelThicknessMM:    18.0,
		trimQualityScorePct: 95.0,
		pressureZoneBar:     3.5,
		preMillingDepthMM:   0.2,
		hotAirTempC:         350.0,
		jointStrengthNCM:    120.0,
		scraperPositionMM:   0.1,
		buffingSpeedRPM:     3000.0,
	}
}

func (m *EdgeBanderMachine) Start(ctx context.Context) {
	go m.updateTags(ctx)
	m.BaseMachine.Start(ctx)
}

func (m *EdgeBanderMachine) updateTags(ctx context.Context) {
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
				m.glueTempC += (rng.Float64()*2 - 1) * 1.0
				clampF(&m.glueTempC, 170.0, 230.0)
				m.feedSpeedMMin += (rng.Float64()*2 - 1) * 0.2
				clampF(&m.feedSpeedMMin, 5.0, 25.0)
				m.trimmerPositionMM += (rng.Float64()*2 - 1) * 0.01
				clampF(&m.trimmerPositionMM, 0.1, 1.5)
				m.tapeTensionN += (rng.Float64()*2 - 1) * 0.5
				clampF(&m.tapeTensionN, 8.0, 40.0)
				m.rollerForceN += (rng.Float64()*2 - 1) * 3.0
				clampF(&m.rollerForceN, 100.0, 500.0)
				m.edgeThicknessMM += (rng.Float64()*2 - 1) * 0.01
				clampF(&m.edgeThicknessMM, 0.4, 3.0)
				m.glueFlowGMin += (rng.Float64()*2 - 1) * 0.5
				clampF(&m.glueFlowGMin, 10.0, 50.0)
				m.panelThicknessMM += (rng.Float64()*2 - 1) * 0.05
				clampF(&m.panelThicknessMM, 8.0, 40.0)
				m.trimQualityScorePct += (rng.Float64()*2 - 1) * 0.3
				clampF(&m.trimQualityScorePct, 75.0, 100.0)
				m.pressureZoneBar += (rng.Float64()*2 - 1) * 0.1
				clampF(&m.pressureZoneBar, 1.5, 6.0)
				m.preMillingDepthMM += (rng.Float64()*2 - 1) * 0.005
				clampF(&m.preMillingDepthMM, 0.05, 0.5)
				m.hotAirTempC += (rng.Float64()*2 - 1) * 2.0
				clampF(&m.hotAirTempC, 280.0, 450.0)
				m.jointStrengthNCM += (rng.Float64()*2 - 1) * 2.0
				clampF(&m.jointStrengthNCM, 60.0, 200.0)
				m.scraperPositionMM += (rng.Float64()*2 - 1) * 0.005
				clampF(&m.scraperPositionMM, 0.0, 0.5)
				m.buffingSpeedRPM += (rng.Float64()*2 - 1) * 20.0
				clampF(&m.buffingSpeedRPM, 1500.0, 5000.0)
			}
			m.Unlock()
		}
	}
}

func (m *EdgeBanderMachine) GetTags() map[string]interface{} {
	m.RLock()
	defer m.RUnlock()
	tags := machine.BaseTagValues(m.BaseMachine)
	tags["glue_temp_c"] = m.glueTempC
	tags["feed_speed_m_min"] = m.feedSpeedMMin
	tags["trimmer_position_mm"] = m.trimmerPositionMM
	tags["tape_tension_n"] = m.tapeTensionN
	tags["roller_force_n"] = m.rollerForceN
	tags["edge_thickness_mm"] = m.edgeThicknessMM
	tags["glue_flow_g_min"] = m.glueFlowGMin
	tags["panel_thickness_mm"] = m.panelThicknessMM
	tags["trim_quality_score_pct"] = m.trimQualityScorePct
	tags["pressure_zone_bar"] = m.pressureZoneBar
	tags["pre_milling_depth_mm"] = m.preMillingDepthMM
	tags["hot_air_temp_c"] = m.hotAirTempC
	tags["joint_strength_n_cm"] = m.jointStrengthNCM
	tags["scraper_position_mm"] = m.scraperPositionMM
	tags["buffing_speed_rpm"] = m.buffingSpeedRPM
	return tags
}

func (m *EdgeBanderMachine) GetTagDefinitions() []machine.TagDefinition {
	defs := machine.BaseTags()
	defs = append(defs,
		machine.TagDefinition{Name: "glue_temp_c", DisplayName: "Glue Temperature (\u00b0C)", NodeID: 7, DataType: "Double", Unit: "°C", Description: "Glue pot temperature"},
		machine.TagDefinition{Name: "feed_speed_m_min", DisplayName: "Feed Speed (m/min)", NodeID: 8, DataType: "Double", Unit: "m/min", Description: "Feed speed"},
		machine.TagDefinition{Name: "trimmer_position_mm", DisplayName: "Trimmer Position (mm)", NodeID: 9, DataType: "Double", Unit: "mm", Description: "Trimmer position"},
		machine.TagDefinition{Name: "tape_tension_n", DisplayName: "Tape Tension (N)", NodeID: 10, DataType: "Double", Unit: "N", Description: "Edge tape tension"},
		machine.TagDefinition{Name: "roller_force_n", DisplayName: "Roller Force (N)", NodeID: 11, DataType: "Double", Unit: "N", Description: "Pressure roller force"},
		machine.TagDefinition{Name: "edge_thickness_mm", DisplayName: "Edge Thickness (mm)", NodeID: 12, DataType: "Double", Unit: "mm", Description: "Edge band thickness"},
		machine.TagDefinition{Name: "glue_flow_g_min", DisplayName: "Glue Flow (g/min)", NodeID: 13, DataType: "Double", Unit: "g/min", Description: "Glue flow rate"},
		machine.TagDefinition{Name: "panel_thickness_mm", DisplayName: "Panel Thickness (mm)", NodeID: 14, DataType: "Double", Unit: "mm", Description: "Panel thickness"},
		machine.TagDefinition{Name: "trim_quality_score_pct", DisplayName: "Trim Quality (%)", NodeID: 15, DataType: "Double", Unit: "%", Description: "Trim quality score"},
		machine.TagDefinition{Name: "pressure_zone_bar", DisplayName: "Pressure Zone (bar)", NodeID: 16, DataType: "Double", Unit: "bar", Description: "Pressure zone setting"},
		machine.TagDefinition{Name: "pre_milling_depth_mm", DisplayName: "Pre-Milling Depth (mm)", NodeID: 17, DataType: "Double", Unit: "mm", Description: "Pre-milling depth"},
		machine.TagDefinition{Name: "hot_air_temp_c", DisplayName: "Hot Air Temp (\u00b0C)", NodeID: 18, DataType: "Double", Unit: "°C", Description: "Hot air temperature"},
		machine.TagDefinition{Name: "joint_strength_n_cm", DisplayName: "Joint Strength (N/cm)", NodeID: 19, DataType: "Double", Unit: "N/cm", Description: "Joint strength"},
		machine.TagDefinition{Name: "scraper_position_mm", DisplayName: "Scraper Position (mm)", NodeID: 20, DataType: "Double", Unit: "mm", Description: "Scraper position"},
		machine.TagDefinition{Name: "buffing_speed_rpm", DisplayName: "Buffing Speed (RPM)", NodeID: 21, DataType: "Double", Unit: "RPM", Description: "Buffing unit speed"},
	)
	return defs
}
