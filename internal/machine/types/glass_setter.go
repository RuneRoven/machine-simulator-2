package types

import (
	"context"
	"time"

	"machine-simulator-2/internal/config"
	"machine-simulator-2/internal/machine"
)

func init() {
	machine.Register("glass_setter", func(cfg config.MachineInstanceConfig) machine.Machine {
		return NewGlassSetter(cfg)
	})
}

// GlassSetterMachine simulates a window glass setting/glazing machine.
type GlassSetterMachine struct {
	*machine.BaseMachine

	suctionVacuumKPa    float64
	glassThicknessMM    float64
	sealantFlowMLMin    float64
	positioningAccMM    float64
	glazingDepthMM      float64
	frameDimensionMM    float64
	glassWeightKg       float64
	sealantTempC        float64
	pressForceN         float64
	rollerSpeedMMS      float64
	gasketCompressionPct float64
	uvCureIntMWCM2      float64
	edgeClearanceMM     float64
	moistureLevelPct    float64
	alignmentOffsetMM   float64
}

func NewGlassSetter(cfg config.MachineInstanceConfig) *GlassSetterMachine {
	return &GlassSetterMachine{
		BaseMachine:          machine.NewBaseMachine(cfg),
		suctionVacuumKPa:     -65.0,
		glassThicknessMM:     4.0,
		sealantFlowMLMin:     25.0,
		positioningAccMM:     0.3,
		glazingDepthMM:       18.0,
		frameDimensionMM:     1200.0,
		glassWeightKg:        15.0,
		sealantTempC:         22.0,
		pressForceN:          200.0,
		rollerSpeedMMS:       80.0,
		gasketCompressionPct: 25.0,
		uvCureIntMWCM2:       120.0,
		edgeClearanceMM:      3.0,
		moistureLevelPct:     8.0,
		alignmentOffsetMM:    0.1,
	}
}

func (m *GlassSetterMachine) Start(ctx context.Context) {
	go m.updateTags(ctx)
	m.BaseMachine.Start(ctx)
}

func (m *GlassSetterMachine) updateTags(ctx context.Context) {
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
				m.suctionVacuumKPa += (rng.Float64()*2 - 1) * 0.5
				clampF(&m.suctionVacuumKPa, -85.0, -30.0)
				m.glassThicknessMM += (rng.Float64()*2 - 1) * 0.01
				clampF(&m.glassThicknessMM, 3.0, 24.0)
				m.sealantFlowMLMin += (rng.Float64()*2 - 1) * 0.5
				clampF(&m.sealantFlowMLMin, 10.0, 50.0)
				m.positioningAccMM += (rng.Float64()*2 - 1) * 0.01
				clampF(&m.positioningAccMM, 0.05, 1.0)
				m.glazingDepthMM += (rng.Float64()*2 - 1) * 0.1
				clampF(&m.glazingDepthMM, 10.0, 30.0)
				m.frameDimensionMM += (rng.Float64()*2 - 1) * 0.5
				clampF(&m.frameDimensionMM, 400.0, 3000.0)
				m.glassWeightKg += (rng.Float64()*2 - 1) * 0.1
				clampF(&m.glassWeightKg, 3.0, 50.0)
				m.sealantTempC += (rng.Float64()*2 - 1) * 0.2
				clampF(&m.sealantTempC, 15.0, 35.0)
				m.pressForceN += (rng.Float64()*2 - 1) * 3.0
				clampF(&m.pressForceN, 50.0, 500.0)
				m.rollerSpeedMMS += (rng.Float64()*2 - 1) * 1.0
				clampF(&m.rollerSpeedMMS, 30.0, 150.0)
				m.gasketCompressionPct += (rng.Float64()*2 - 1) * 0.3
				clampF(&m.gasketCompressionPct, 10.0, 50.0)
				m.uvCureIntMWCM2 += (rng.Float64()*2 - 1) * 2.0
				clampF(&m.uvCureIntMWCM2, 50.0, 250.0)
				m.edgeClearanceMM += (rng.Float64()*2 - 1) * 0.05
				clampF(&m.edgeClearanceMM, 1.0, 6.0)
				m.moistureLevelPct += (rng.Float64()*2 - 1) * 0.2
				clampF(&m.moistureLevelPct, 2.0, 20.0)
				m.alignmentOffsetMM += (rng.Float64()*2 - 1) * 0.02
				clampF(&m.alignmentOffsetMM, 0.0, 1.0)
			}
			m.Unlock()
		}
	}
}

func (m *GlassSetterMachine) GetTags() map[string]interface{} {
	m.RLock()
	defer m.RUnlock()
	tags := machine.BaseTagValues(m.BaseMachine)
	tags["suction_vacuum_kpa"] = m.suctionVacuumKPa
	tags["glass_thickness_mm"] = m.glassThicknessMM
	tags["sealant_flow_ml_min"] = m.sealantFlowMLMin
	tags["positioning_accuracy_mm"] = m.positioningAccMM
	tags["glazing_depth_mm"] = m.glazingDepthMM
	tags["frame_dimension_mm"] = m.frameDimensionMM
	tags["glass_weight_kg"] = m.glassWeightKg
	tags["sealant_temp_c"] = m.sealantTempC
	tags["press_force_n"] = m.pressForceN
	tags["roller_speed_mm_s"] = m.rollerSpeedMMS
	tags["gasket_compression_pct"] = m.gasketCompressionPct
	tags["uv_cure_intensity_mw_cm2"] = m.uvCureIntMWCM2
	tags["edge_clearance_mm"] = m.edgeClearanceMM
	tags["moisture_level_pct"] = m.moistureLevelPct
	tags["alignment_offset_mm"] = m.alignmentOffsetMM
	return tags
}

func (m *GlassSetterMachine) GetTagDefinitions() []machine.TagDefinition {
	defs := machine.BaseTags()
	defs = append(defs,
		machine.TagDefinition{Name: "suction_vacuum_kpa", DisplayName: "Suction Vacuum (kPa)", NodeID: 7, DataType: "Double", Unit: "kPa", Description: "Suction cup vacuum pressure"},
		machine.TagDefinition{Name: "glass_thickness_mm", DisplayName: "Glass Thickness (mm)", NodeID: 8, DataType: "Double", Unit: "mm", Description: "Glass thickness"},
		machine.TagDefinition{Name: "sealant_flow_ml_min", DisplayName: "Sealant Flow (mL/min)", NodeID: 9, DataType: "Double", Unit: "mL/min", Description: "Sealant flow rate"},
		machine.TagDefinition{Name: "positioning_accuracy_mm", DisplayName: "Positioning Accuracy (mm)", NodeID: 10, DataType: "Double", Unit: "mm", Description: "Glass positioning accuracy"},
		machine.TagDefinition{Name: "glazing_depth_mm", DisplayName: "Glazing Depth (mm)", NodeID: 11, DataType: "Double", Unit: "mm", Description: "Glazing depth"},
		machine.TagDefinition{Name: "frame_dimension_mm", DisplayName: "Frame Dimension (mm)", NodeID: 12, DataType: "Double", Unit: "mm", Description: "Frame dimension"},
		machine.TagDefinition{Name: "glass_weight_kg", DisplayName: "Glass Weight (kg)", NodeID: 13, DataType: "Double", Unit: "kg", Description: "Glass weight"},
		machine.TagDefinition{Name: "sealant_temp_c", DisplayName: "Sealant Temp (°C)", NodeID: 14, DataType: "Double", Unit: "°C", Description: "Sealant temperature"},
		machine.TagDefinition{Name: "press_force_n", DisplayName: "Press Force (N)", NodeID: 15, DataType: "Double", Unit: "N", Description: "Glass press force"},
		machine.TagDefinition{Name: "roller_speed_mm_s", DisplayName: "Roller Speed (mm/s)", NodeID: 16, DataType: "Double", Unit: "mm/s", Description: "Glazing roller speed"},
		machine.TagDefinition{Name: "gasket_compression_pct", DisplayName: "Gasket Compression (%)", NodeID: 17, DataType: "Double", Unit: "%", Description: "Gasket compression"},
		machine.TagDefinition{Name: "uv_cure_intensity_mw_cm2", DisplayName: "UV Cure Intensity (mW/cm²)", NodeID: 18, DataType: "Double", Unit: "mW/cm²", Description: "UV cure intensity"},
		machine.TagDefinition{Name: "edge_clearance_mm", DisplayName: "Edge Clearance (mm)", NodeID: 19, DataType: "Double", Unit: "mm", Description: "Glass edge clearance"},
		machine.TagDefinition{Name: "moisture_level_pct", DisplayName: "Moisture Level (%)", NodeID: 20, DataType: "Double", Unit: "%", Description: "Moisture level"},
		machine.TagDefinition{Name: "alignment_offset_mm", DisplayName: "Alignment Offset (mm)", NodeID: 21, DataType: "Double", Unit: "mm", Description: "Alignment offset"},
	)
	return defs
}
