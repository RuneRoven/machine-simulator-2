package types

import (
	"context"
	"time"

	"machine-simulator-2/internal/config"
	"machine-simulator-2/internal/machine"
)

func init() {
	machine.Register("wave_solder", func(cfg config.MachineInstanceConfig) machine.Machine {
		return NewWaveSolder(cfg)
	})
}

// WaveSolderMachine simulates a wave soldering machine.
type WaveSolderMachine struct {
	*machine.BaseMachine

	solderPotTempC    float64
	waveHeightMM      float64
	fluxDensityMLM2   float64
	conveyorSpeedMMin float64
	preheatTopTempC   float64
	preheatBotTempC   float64
	solderPurityPct   float64
	drossLevelPct     float64
	fingerClearanceMM float64
	contactTimeS      float64
	boardAngleDeg     float64
	nitrogenFlowLMin  float64
	potLevelMM        float64
	bridgeDefectCount float64
	icicleCount       float64
}

func NewWaveSolder(cfg config.MachineInstanceConfig) *WaveSolderMachine {
	return &WaveSolderMachine{
		BaseMachine:       machine.NewBaseMachine(cfg),
		solderPotTempC:    255.0,
		waveHeightMM:      5.0,
		fluxDensityMLM2:   80.0,
		conveyorSpeedMMin: 1.2,
		preheatTopTempC:   110.0,
		preheatBotTempC:   95.0,
		solderPurityPct:   99.5,
		drossLevelPct:     3.0,
		fingerClearanceMM: 2.0,
		contactTimeS:      3.0,
		boardAngleDeg:     6.0,
		nitrogenFlowLMin:  30.0,
		potLevelMM:        180.0,
		bridgeDefectCount: 0.0,
		icicleCount:       0.0,
	}
}

func (m *WaveSolderMachine) Start(ctx context.Context) {
	go m.updateTags(ctx)
	m.BaseMachine.Start(ctx)
}

func (m *WaveSolderMachine) updateTags(ctx context.Context) {
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
				m.solderPotTempC += (rng.Float64()*2 - 1) * 0.3
				clampF(&m.solderPotTempC, 245.0, 265.0)
				m.waveHeightMM += (rng.Float64()*2 - 1) * 0.1
				clampF(&m.waveHeightMM, 2.0, 10.0)
				m.fluxDensityMLM2 += (rng.Float64()*2 - 1) * 1.0
				clampF(&m.fluxDensityMLM2, 40.0, 150.0)
				m.conveyorSpeedMMin += (rng.Float64()*2 - 1) * 0.02
				clampF(&m.conveyorSpeedMMin, 0.5, 2.5)
				m.preheatTopTempC += (rng.Float64()*2 - 1) * 0.5
				clampF(&m.preheatTopTempC, 80.0, 140.0)
				m.preheatBotTempC += (rng.Float64()*2 - 1) * 0.5
				clampF(&m.preheatBotTempC, 60.0, 120.0)
				m.solderPurityPct -= rng.Float64() * 0.001
				clampF(&m.solderPurityPct, 97.0, 99.99)
				m.drossLevelPct += rng.Float64() * 0.005
				clampF(&m.drossLevelPct, 0.0, 15.0)
				m.fingerClearanceMM += (rng.Float64()*2 - 1) * 0.05
				clampF(&m.fingerClearanceMM, 0.5, 5.0)
				m.contactTimeS += (rng.Float64()*2 - 1) * 0.05
				clampF(&m.contactTimeS, 1.0, 6.0)
				m.boardAngleDeg += (rng.Float64()*2 - 1) * 0.1
				clampF(&m.boardAngleDeg, 3.0, 10.0)
				m.nitrogenFlowLMin += (rng.Float64()*2 - 1) * 0.5
				clampF(&m.nitrogenFlowLMin, 10.0, 60.0)
				m.potLevelMM += (rng.Float64()*2 - 1) * 0.2
				clampF(&m.potLevelMM, 150.0, 220.0)
				if rng.Float64() < 0.005 {
					m.bridgeDefectCount++
				}
				if rng.Float64() < 0.003 {
					m.icicleCount++
				}
			}
			m.Unlock()
		}
	}
}

func (m *WaveSolderMachine) GetTags() map[string]interface{} {
	m.RLock()
	defer m.RUnlock()
	tags := machine.BaseTagValues(m.BaseMachine)
	tags["solder_pot_temp_c"] = m.solderPotTempC
	tags["wave_height_mm"] = m.waveHeightMM
	tags["flux_density_ml_m2"] = m.fluxDensityMLM2
	tags["conveyor_speed_m_min"] = m.conveyorSpeedMMin
	tags["preheat_top_temp_c"] = m.preheatTopTempC
	tags["preheat_bottom_temp_c"] = m.preheatBotTempC
	tags["solder_purity_pct"] = m.solderPurityPct
	tags["dross_level_pct"] = m.drossLevelPct
	tags["finger_clearance_mm"] = m.fingerClearanceMM
	tags["contact_time_s"] = m.contactTimeS
	tags["board_angle_deg"] = m.boardAngleDeg
	tags["nitrogen_flow_l_min"] = m.nitrogenFlowLMin
	tags["pot_level_mm"] = m.potLevelMM
	tags["bridge_defect_count"] = m.bridgeDefectCount
	tags["icicle_count"] = m.icicleCount
	return tags
}

func (m *WaveSolderMachine) GetTagDefinitions() []machine.TagDefinition {
	defs := machine.BaseTags()
	defs = append(defs,
		machine.TagDefinition{Name: "solder_pot_temp_c", DisplayName: "Solder Pot Temp (\u00b0C)", NodeID: 7, DataType: "Double", Unit: "°C", Description: "Solder pot temperature"},
		machine.TagDefinition{Name: "wave_height_mm", DisplayName: "Wave Height (mm)", NodeID: 8, DataType: "Double", Unit: "mm", Description: "Solder wave height"},
		machine.TagDefinition{Name: "flux_density_ml_m2", DisplayName: "Flux Density (mL/m\u00b2)", NodeID: 9, DataType: "Double", Unit: "mL/m²", Description: "Flux density"},
		machine.TagDefinition{Name: "conveyor_speed_m_min", DisplayName: "Conveyor Speed (m/min)", NodeID: 10, DataType: "Double", Unit: "m/min", Description: "Conveyor speed"},
		machine.TagDefinition{Name: "preheat_top_temp_c", DisplayName: "Top Preheat Temp (\u00b0C)", NodeID: 11, DataType: "Double", Unit: "°C", Description: "Top preheat temperature"},
		machine.TagDefinition{Name: "preheat_bottom_temp_c", DisplayName: "Bottom Preheat Temp (\u00b0C)", NodeID: 12, DataType: "Double", Unit: "°C", Description: "Bottom preheat temperature"},
		machine.TagDefinition{Name: "solder_purity_pct", DisplayName: "Solder Purity (%)", NodeID: 13, DataType: "Double", Unit: "%", Description: "Solder purity level"},
		machine.TagDefinition{Name: "dross_level_pct", DisplayName: "Dross Level (%)", NodeID: 14, DataType: "Double", Unit: "%", Description: "Dross accumulation level"},
		machine.TagDefinition{Name: "finger_clearance_mm", DisplayName: "Finger Clearance (mm)", NodeID: 15, DataType: "Double", Unit: "mm", Description: "Solder finger clearance"},
		machine.TagDefinition{Name: "contact_time_s", DisplayName: "Contact Time (s)", NodeID: 16, DataType: "Double", Unit: "s", Description: "Board-wave contact time"},
		machine.TagDefinition{Name: "board_angle_deg", DisplayName: "Board Angle (\u00b0)", NodeID: 17, DataType: "Double", Unit: "°", Description: "Board conveyor angle"},
		machine.TagDefinition{Name: "nitrogen_flow_l_min", DisplayName: "Nitrogen Flow (L/min)", NodeID: 18, DataType: "Double", Unit: "L/min", Description: "Nitrogen flow rate"},
		machine.TagDefinition{Name: "pot_level_mm", DisplayName: "Pot Level (mm)", NodeID: 19, DataType: "Double", Unit: "mm", Description: "Solder pot level"},
		machine.TagDefinition{Name: "bridge_defect_count", DisplayName: "Bridge Defect Count", NodeID: 20, DataType: "Double", Unit: "", Description: "Solder bridge defect count"},
		machine.TagDefinition{Name: "icicle_count", DisplayName: "Icicle Defect Count", NodeID: 21, DataType: "Double", Unit: "", Description: "Solder icicle defect count"},
	)
	return defs
}
