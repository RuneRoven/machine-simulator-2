package types

import (
	"context"
	"time"

	"machine-simulator-2/internal/config"
	"machine-simulator-2/internal/machine"
)

func init() {
	machine.Register("spot_welder", func(cfg config.MachineInstanceConfig) machine.Machine {
		return NewSpotWelder(cfg)
	})
}

// SpotWelderMachine simulates a resistance spot welding machine.
type SpotWelderMachine struct {
	*machine.BaseMachine

	weldCurrentKA    float64
	electrodeForceKN float64
	weldTimeMS       float64
	nuggetDiamMM     float64
	tipWearPct       float64
	squeezeTimeMS    float64
	holdTimeMS       float64
	electrodeTempC   float64
	waterFlowLMin    float64
	resistanceMOhm   float64
	expulsionCount   float64
	weldPowerKW      float64
	capDisplaceMM    float64
	shuntFactor      float64
	sheetGapMM       float64
}

func NewSpotWelder(cfg config.MachineInstanceConfig) *SpotWelderMachine {
	return &SpotWelderMachine{
		BaseMachine:      machine.NewBaseMachine(cfg),
		weldCurrentKA:    10.5,
		electrodeForceKN: 3.2,
		weldTimeMS:       200.0,
		nuggetDiamMM:     5.5,
		tipWearPct:       3.0,
		squeezeTimeMS:    100.0,
		holdTimeMS:       150.0,
		electrodeTempC:   85.0,
		waterFlowLMin:    6.0,
		resistanceMOhm:   120.0,
		expulsionCount:   0.0,
		weldPowerKW:      45.0,
		capDisplaceMM:    0.3,
		shuntFactor:      0.92,
		sheetGapMM:       0.05,
	}
}

func (m *SpotWelderMachine) Start(ctx context.Context) {
	go m.updateTags(ctx)
	m.BaseMachine.Start(ctx)
}

func (m *SpotWelderMachine) updateTags(ctx context.Context) {
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
				m.weldCurrentKA += (rng.Float64()*2 - 1) * 0.2
				clampF(&m.weldCurrentKA, 8.0, 14.0)
				m.electrodeForceKN += (rng.Float64()*2 - 1) * 0.1
				clampF(&m.electrodeForceKN, 2.0, 5.0)
				m.weldTimeMS += (rng.Float64()*2 - 1) * 5.0
				clampF(&m.weldTimeMS, 100.0, 400.0)
				m.nuggetDiamMM += (rng.Float64()*2 - 1) * 0.1
				clampF(&m.nuggetDiamMM, 4.0, 7.0)
				m.tipWearPct += rng.Float64() * 0.005
				clampF(&m.tipWearPct, 0.0, 100.0)
				m.squeezeTimeMS += (rng.Float64()*2 - 1) * 2.0
				clampF(&m.squeezeTimeMS, 50.0, 200.0)
				m.holdTimeMS += (rng.Float64()*2 - 1) * 2.0
				clampF(&m.holdTimeMS, 80.0, 300.0)
				m.electrodeTempC += (rng.Float64()*2 - 1) * 1.0
				clampF(&m.electrodeTempC, 40.0, 200.0)
				m.waterFlowLMin += (rng.Float64()*2 - 1) * 0.1
				clampF(&m.waterFlowLMin, 3.0, 10.0)
				m.resistanceMOhm += (rng.Float64()*2 - 1) * 2.0
				clampF(&m.resistanceMOhm, 50.0, 250.0)
				if rng.Float64() < 0.01 {
					m.expulsionCount++
				}
				m.weldPowerKW += (rng.Float64()*2 - 1) * 1.0
				clampF(&m.weldPowerKW, 20.0, 80.0)
				m.capDisplaceMM += (rng.Float64()*2 - 1) * 0.02
				clampF(&m.capDisplaceMM, 0.1, 0.8)
				m.shuntFactor += (rng.Float64()*2 - 1) * 0.005
				clampF(&m.shuntFactor, 0.80, 1.0)
				m.sheetGapMM += (rng.Float64()*2 - 1) * 0.01
				clampF(&m.sheetGapMM, 0.0, 0.5)
			}
			m.Unlock()
		}
	}
}

func (m *SpotWelderMachine) GetTags() map[string]interface{} {
	m.RLock()
	defer m.RUnlock()
	tags := machine.BaseTagValues(m.BaseMachine)
	tags["weld_current_ka"] = m.weldCurrentKA
	tags["electrode_force_kn"] = m.electrodeForceKN
	tags["weld_time_ms"] = m.weldTimeMS
	tags["nugget_diameter_mm"] = m.nuggetDiamMM
	tags["tip_wear_pct"] = m.tipWearPct
	tags["squeeze_time_ms"] = m.squeezeTimeMS
	tags["hold_time_ms"] = m.holdTimeMS
	tags["electrode_temp_c"] = m.electrodeTempC
	tags["water_flow_l_min"] = m.waterFlowLMin
	tags["resistance_mohm"] = m.resistanceMOhm
	tags["expulsion_count"] = m.expulsionCount
	tags["weld_power_kw"] = m.weldPowerKW
	tags["cap_displacement_mm"] = m.capDisplaceMM
	tags["shunt_factor"] = m.shuntFactor
	tags["sheet_gap_mm"] = m.sheetGapMM
	return tags
}

func (m *SpotWelderMachine) GetTagDefinitions() []machine.TagDefinition {
	defs := machine.BaseTags()
	defs = append(defs,
		machine.TagDefinition{Name: "weld_current_ka", DisplayName: "Welding Current (kA)", NodeID: 7, DataType: "Double", Unit: "kA", Description: "Welding current"},
		machine.TagDefinition{Name: "electrode_force_kn", DisplayName: "Electrode Force (kN)", NodeID: 8, DataType: "Double", Unit: "kN", Description: "Electrode force"},
		machine.TagDefinition{Name: "weld_time_ms", DisplayName: "Weld Duration (ms)", NodeID: 9, DataType: "Double", Unit: "ms", Description: "Weld duration"},
		machine.TagDefinition{Name: "nugget_diameter_mm", DisplayName: "Nugget Diameter (mm)", NodeID: 10, DataType: "Double", Unit: "mm", Description: "Weld nugget diameter"},
		machine.TagDefinition{Name: "tip_wear_pct", DisplayName: "Tip Wear (%)", NodeID: 11, DataType: "Double", Unit: "%", Description: "Electrode tip wear"},
		machine.TagDefinition{Name: "squeeze_time_ms", DisplayName: "Squeeze Time (ms)", NodeID: 12, DataType: "Double", Unit: "ms", Description: "Squeeze time before weld"},
		machine.TagDefinition{Name: "hold_time_ms", DisplayName: "Hold Time (ms)", NodeID: 13, DataType: "Double", Unit: "ms", Description: "Hold time after weld"},
		machine.TagDefinition{Name: "electrode_temp_c", DisplayName: "Electrode Temp (\u00b0C)", NodeID: 14, DataType: "Double", Unit: "°C", Description: "Electrode temperature"},
		machine.TagDefinition{Name: "water_flow_l_min", DisplayName: "Water Flow Rate (L/min)", NodeID: 15, DataType: "Double", Unit: "L/min", Description: "Cooling water flow rate"},
		machine.TagDefinition{Name: "resistance_mohm", DisplayName: "Contact Resistance (m\u03a9)", NodeID: 16, DataType: "Double", Unit: "mΩ", Description: "Contact resistance"},
		machine.TagDefinition{Name: "expulsion_count", DisplayName: "Expulsion Count", NodeID: 17, DataType: "Double", Unit: "", Description: "Expulsion event count"},
		machine.TagDefinition{Name: "weld_power_kw", DisplayName: "Welding Power (kW)", NodeID: 18, DataType: "Double", Unit: "kW", Description: "Welding power"},
		machine.TagDefinition{Name: "cap_displacement_mm", DisplayName: "Cap Displacement (mm)", NodeID: 19, DataType: "Double", Unit: "mm", Description: "Cap tip displacement"},
		machine.TagDefinition{Name: "shunt_factor", DisplayName: "Shunt Factor", NodeID: 20, DataType: "Double", Unit: "", Description: "Current shunt factor"},
		machine.TagDefinition{Name: "sheet_gap_mm", DisplayName: "Sheet Gap (mm)", NodeID: 21, DataType: "Double", Unit: "mm", Description: "Gap between sheets"},
	)
	return defs
}
