package types

import (
	"context"
	"time"

	"machine-simulator-2/internal/config"
	"machine-simulator-2/internal/machine"
)

func init() {
	machine.Register("injection_molding", func(cfg config.MachineInstanceConfig) machine.Machine {
		return NewInjectionMolding(cfg)
	})
}

// InjectionMoldingMachine simulates a plastic injection molding machine.
type InjectionMoldingMachine struct {
	*machine.BaseMachine

	barrelZone1TempC   float64
	barrelZone2TempC   float64
	barrelZone3TempC   float64
	barrelZone4TempC   float64
	injectionPressBar  float64
	moldTempC          float64
	clampForceTons     float64
	coolingTimeS       float64
	shotWeightG        float64
	cushionMM          float64
	screwSpeedRPM      float64
	backPressureBar    float64
	meltTempC          float64
	packPressureBar    float64
	fillTimeS          float64
}

func NewInjectionMolding(cfg config.MachineInstanceConfig) *InjectionMoldingMachine {
	return &InjectionMoldingMachine{
		BaseMachine:       machine.NewBaseMachine(cfg),
		barrelZone1TempC:  180.0,
		barrelZone2TempC:  200.0,
		barrelZone3TempC:  215.0,
		barrelZone4TempC:  225.0,
		injectionPressBar: 1200.0,
		moldTempC:         45.0,
		clampForceTons:    200.0,
		coolingTimeS:      12.0,
		shotWeightG:       85.0,
		cushionMM:         5.0,
		screwSpeedRPM:     100.0,
		backPressureBar:   30.0,
		meltTempC:         230.0,
		packPressureBar:   800.0,
		fillTimeS:         1.5,
	}
}

func (m *InjectionMoldingMachine) Start(ctx context.Context) {
	go m.updateTags(ctx)
	m.BaseMachine.Start(ctx)
}

func (m *InjectionMoldingMachine) updateTags(ctx context.Context) {
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
				m.barrelZone1TempC += (rng.Float64()*2 - 1) * 0.3
				clampF(&m.barrelZone1TempC, 160.0, 200.0)
				m.barrelZone2TempC += (rng.Float64()*2 - 1) * 0.3
				clampF(&m.barrelZone2TempC, 180.0, 220.0)
				m.barrelZone3TempC += (rng.Float64()*2 - 1) * 0.3
				clampF(&m.barrelZone3TempC, 200.0, 240.0)
				m.barrelZone4TempC += (rng.Float64()*2 - 1) * 0.3
				clampF(&m.barrelZone4TempC, 210.0, 250.0)
				m.injectionPressBar += (rng.Float64()*2 - 1) * 10.0
				clampF(&m.injectionPressBar, 500.0, 2000.0)
				m.moldTempC += (rng.Float64()*2 - 1) * 0.2
				clampF(&m.moldTempC, 20.0, 80.0)
				m.clampForceTons += (rng.Float64()*2 - 1) * 2.0
				clampF(&m.clampForceTons, 50.0, 500.0)
				m.coolingTimeS += (rng.Float64()*2 - 1) * 0.1
				clampF(&m.coolingTimeS, 5.0, 30.0)
				m.shotWeightG += (rng.Float64()*2 - 1) * 0.3
				clampF(&m.shotWeightG, 20.0, 200.0)
				m.cushionMM += (rng.Float64()*2 - 1) * 0.1
				clampF(&m.cushionMM, 2.0, 12.0)
				m.screwSpeedRPM += (rng.Float64()*2 - 1) * 1.0
				clampF(&m.screwSpeedRPM, 30.0, 200.0)
				m.backPressureBar += (rng.Float64()*2 - 1) * 0.5
				clampF(&m.backPressureBar, 10.0, 80.0)
				m.meltTempC += (rng.Float64()*2 - 1) * 0.3
				clampF(&m.meltTempC, 200.0, 280.0)
				m.packPressureBar += (rng.Float64()*2 - 1) * 5.0
				clampF(&m.packPressureBar, 300.0, 1500.0)
				m.fillTimeS += (rng.Float64()*2 - 1) * 0.02
				clampF(&m.fillTimeS, 0.5, 5.0)
			}
			m.Unlock()
		}
	}
}

func (m *InjectionMoldingMachine) GetTags() map[string]interface{} {
	m.RLock()
	defer m.RUnlock()
	tags := machine.BaseTagValues(m.BaseMachine)
	tags["barrel_zone1_temp_c"] = m.barrelZone1TempC
	tags["barrel_zone2_temp_c"] = m.barrelZone2TempC
	tags["barrel_zone3_temp_c"] = m.barrelZone3TempC
	tags["barrel_zone4_temp_c"] = m.barrelZone4TempC
	tags["injection_pressure_bar"] = m.injectionPressBar
	tags["mold_temp_c"] = m.moldTempC
	tags["clamp_force_tons"] = m.clampForceTons
	tags["cooling_time_s"] = m.coolingTimeS
	tags["shot_weight_g"] = m.shotWeightG
	tags["cushion_mm"] = m.cushionMM
	tags["screw_speed_rpm"] = m.screwSpeedRPM
	tags["back_pressure_bar"] = m.backPressureBar
	tags["melt_temp_c"] = m.meltTempC
	tags["pack_pressure_bar"] = m.packPressureBar
	tags["fill_time_s"] = m.fillTimeS
	return tags
}

func (m *InjectionMoldingMachine) GetTagDefinitions() []machine.TagDefinition {
	defs := machine.BaseTags()
	defs = append(defs,
		machine.TagDefinition{Name: "barrel_zone1_temp_c", DisplayName: "Barrel Zone 1 Temp (\u00b0C)", NodeID: 7, DataType: "Double", Unit: "°C", Description: "Barrel zone 1 temperature (feed)"},
		machine.TagDefinition{Name: "barrel_zone2_temp_c", DisplayName: "Barrel Zone 2 Temp (\u00b0C)", NodeID: 8, DataType: "Double", Unit: "°C", Description: "Barrel zone 2 temperature"},
		machine.TagDefinition{Name: "barrel_zone3_temp_c", DisplayName: "Barrel Zone 3 Temp (\u00b0C)", NodeID: 9, DataType: "Double", Unit: "°C", Description: "Barrel zone 3 temperature"},
		machine.TagDefinition{Name: "barrel_zone4_temp_c", DisplayName: "Barrel Zone 4 Temp (\u00b0C)", NodeID: 10, DataType: "Double", Unit: "°C", Description: "Barrel zone 4 temperature (nozzle)"},
		machine.TagDefinition{Name: "injection_pressure_bar", DisplayName: "Injection Pressure (bar)", NodeID: 11, DataType: "Double", Unit: "bar", Description: "Injection pressure"},
		machine.TagDefinition{Name: "mold_temp_c", DisplayName: "Mold Temperature (\u00b0C)", NodeID: 12, DataType: "Double", Unit: "°C", Description: "Mold temperature"},
		machine.TagDefinition{Name: "clamp_force_tons", DisplayName: "Clamp Force (tons)", NodeID: 13, DataType: "Double", Unit: "tons", Description: "Clamp force"},
		machine.TagDefinition{Name: "cooling_time_s", DisplayName: "Cooling Time (s)", NodeID: 14, DataType: "Double", Unit: "s", Description: "Cooling time"},
		machine.TagDefinition{Name: "shot_weight_g", DisplayName: "Shot Weight (g)", NodeID: 15, DataType: "Double", Unit: "g", Description: "Shot weight"},
		machine.TagDefinition{Name: "cushion_mm", DisplayName: "Screw Cushion (mm)", NodeID: 16, DataType: "Double", Unit: "mm", Description: "Screw cushion"},
		machine.TagDefinition{Name: "screw_speed_rpm", DisplayName: "Screw Speed (RPM)", NodeID: 17, DataType: "Double", Unit: "RPM", Description: "Screw rotation speed"},
		machine.TagDefinition{Name: "back_pressure_bar", DisplayName: "Back Pressure (bar)", NodeID: 18, DataType: "Double", Unit: "bar", Description: "Back pressure"},
		machine.TagDefinition{Name: "melt_temp_c", DisplayName: "Melt Temperature (\u00b0C)", NodeID: 19, DataType: "Double", Unit: "°C", Description: "Melt temperature"},
		machine.TagDefinition{Name: "pack_pressure_bar", DisplayName: "Pack Pressure (bar)", NodeID: 20, DataType: "Double", Unit: "bar", Description: "Packing pressure"},
		machine.TagDefinition{Name: "fill_time_s", DisplayName: "Fill Time (s)", NodeID: 21, DataType: "Double", Unit: "s", Description: "Mold fill time"},
	)
	return defs
}
