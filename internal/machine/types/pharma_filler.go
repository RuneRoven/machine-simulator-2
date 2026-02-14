package types

import (
	"context"
	"time"

	"machine-simulator-2/internal/config"
	"machine-simulator-2/internal/machine"
)

func init() {
	machine.Register("pharma_filler", func(cfg config.MachineInstanceConfig) machine.Machine {
		return NewPharmaFiller(cfg)
	})
}

// PharmaFillerMachine simulates a pharmaceutical aseptic vial filler.
type PharmaFillerMachine struct {
	*machine.BaseMachine

	fillVolumeML        float64
	fillAccuracyPct     float64
	particulateCount    float64
	cleanRoomPressPA    float64
	laminarFlowVelMS    float64
	stopperForceN       float64
	vialCount           float64
	rejectCount         float64
	fillWeightMG        float64
	tareWeightMG        float64
	nitrogenFlowLMin    float64
	productTempC        float64
	pumpSpeedRPM        float64
	dosingAccuracyPct   float64
	checkweighDevMG     float64
}

func NewPharmaFiller(cfg config.MachineInstanceConfig) *PharmaFillerMachine {
	return &PharmaFillerMachine{
		BaseMachine:      machine.NewBaseMachine(cfg),
		fillVolumeML:     2.0,
		fillAccuracyPct:  99.8,
		particulateCount: 5.0,
		cleanRoomPressPA: 30.0,
		laminarFlowVelMS: 0.45,
		stopperForceN:    35.0,
		vialCount:        0.0,
		rejectCount:      0.0,
		fillWeightMG:     2010.0,
		tareWeightMG:     8500.0,
		nitrogenFlowLMin: 5.0,
		productTempC:     5.0,
		pumpSpeedRPM:     60.0,
		dosingAccuracyPct: 99.5,
		checkweighDevMG:  2.0,
	}
}

func (m *PharmaFillerMachine) Start(ctx context.Context) {
	go m.updateTags(ctx)
	m.BaseMachine.Start(ctx)
}

func (m *PharmaFillerMachine) updateTags(ctx context.Context) {
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
				m.fillVolumeML += (rng.Float64()*2 - 1) * 0.005
				clampF(&m.fillVolumeML, 1.8, 2.2)
				m.fillAccuracyPct += (rng.Float64()*2 - 1) * 0.02
				clampF(&m.fillAccuracyPct, 98.0, 100.0)
				m.particulateCount += (rng.Float64()*2 - 1) * 0.5
				clampF(&m.particulateCount, 0.0, 50.0)
				m.cleanRoomPressPA += (rng.Float64()*2 - 1) * 0.5
				clampF(&m.cleanRoomPressPA, 15.0, 60.0)
				m.laminarFlowVelMS += (rng.Float64()*2 - 1) * 0.005
				clampF(&m.laminarFlowVelMS, 0.36, 0.54)
				m.stopperForceN += (rng.Float64()*2 - 1) * 0.5
				clampF(&m.stopperForceN, 20.0, 60.0)
				m.vialCount += rng.Float64() * 0.5
				if rng.Float64() < 0.002 {
					m.rejectCount++
				}
				m.fillWeightMG += (rng.Float64()*2 - 1) * 1.0
				clampF(&m.fillWeightMG, 1900.0, 2100.0)
				m.tareWeightMG += (rng.Float64()*2 - 1) * 5.0
				clampF(&m.tareWeightMG, 8000.0, 9000.0)
				m.nitrogenFlowLMin += (rng.Float64()*2 - 1) * 0.1
				clampF(&m.nitrogenFlowLMin, 2.0, 10.0)
				m.productTempC += (rng.Float64()*2 - 1) * 0.1
				clampF(&m.productTempC, 2.0, 8.0)
				m.pumpSpeedRPM += (rng.Float64()*2 - 1) * 0.5
				clampF(&m.pumpSpeedRPM, 20.0, 120.0)
				m.dosingAccuracyPct += (rng.Float64()*2 - 1) * 0.02
				clampF(&m.dosingAccuracyPct, 98.0, 100.0)
				m.checkweighDevMG += (rng.Float64()*2 - 1) * 0.2
				clampF(&m.checkweighDevMG, 0.5, 10.0)
			}
			m.Unlock()
		}
	}
}

func (m *PharmaFillerMachine) GetTags() map[string]interface{} {
	m.RLock()
	defer m.RUnlock()
	tags := machine.BaseTagValues(m.BaseMachine)
	tags["fill_volume_ml"] = m.fillVolumeML
	tags["fill_accuracy_pct"] = m.fillAccuracyPct
	tags["particulate_count"] = m.particulateCount
	tags["clean_room_pressure_pa"] = m.cleanRoomPressPA
	tags["laminar_flow_velocity_m_s"] = m.laminarFlowVelMS
	tags["stopper_insertion_force_n"] = m.stopperForceN
	tags["vial_count"] = m.vialCount
	tags["reject_count"] = m.rejectCount
	tags["fill_weight_mg"] = m.fillWeightMG
	tags["tare_weight_mg"] = m.tareWeightMG
	tags["nitrogen_overlay_flow_l_min"] = m.nitrogenFlowLMin
	tags["product_temp_c"] = m.productTempC
	tags["pump_speed_rpm"] = m.pumpSpeedRPM
	tags["dosing_accuracy_pct"] = m.dosingAccuracyPct
	tags["checkweigh_deviation_mg"] = m.checkweighDevMG
	return tags
}

func (m *PharmaFillerMachine) GetTagDefinitions() []machine.TagDefinition {
	defs := machine.BaseTags()
	defs = append(defs,
		machine.TagDefinition{Name: "fill_volume_ml", DisplayName: "Fill Volume (mL)", NodeID: 7, DataType: "Double", Unit: "mL", Description: "Fill volume per vial"},
		machine.TagDefinition{Name: "fill_accuracy_pct", DisplayName: "Fill Accuracy (%)", NodeID: 8, DataType: "Double", Unit: "%", Description: "Fill accuracy"},
		machine.TagDefinition{Name: "particulate_count", DisplayName: "Particulate Count", NodeID: 9, DataType: "Double", Unit: "", Description: "Particulate count per sample"},
		machine.TagDefinition{Name: "clean_room_pressure_pa", DisplayName: "Clean Room Pressure (Pa)", NodeID: 10, DataType: "Double", Unit: "Pa", Description: "Clean room differential pressure"},
		machine.TagDefinition{Name: "laminar_flow_velocity_m_s", DisplayName: "Laminar Flow Velocity (m/s)", NodeID: 11, DataType: "Double", Unit: "m/s", Description: "Laminar airflow velocity"},
		machine.TagDefinition{Name: "stopper_insertion_force_n", DisplayName: "Stopper Force (N)", NodeID: 12, DataType: "Double", Unit: "N", Description: "Stopper insertion force"},
		machine.TagDefinition{Name: "vial_count", DisplayName: "Vial Count", NodeID: 13, DataType: "Double", Unit: "", Description: "Vials filled"},
		machine.TagDefinition{Name: "reject_count", DisplayName: "Reject Count", NodeID: 14, DataType: "Double", Unit: "", Description: "Rejected vials"},
		machine.TagDefinition{Name: "fill_weight_mg", DisplayName: "Fill Weight (mg)", NodeID: 15, DataType: "Double", Unit: "mg", Description: "Measured fill weight"},
		machine.TagDefinition{Name: "tare_weight_mg", DisplayName: "Tare Weight (mg)", NodeID: 16, DataType: "Double", Unit: "mg", Description: "Tare weight of empty vial"},
		machine.TagDefinition{Name: "nitrogen_overlay_flow_l_min", DisplayName: "Nitrogen Flow (L/min)", NodeID: 17, DataType: "Double", Unit: "L/min", Description: "Nitrogen overlay flow"},
		machine.TagDefinition{Name: "product_temp_c", DisplayName: "Product Temp (°C)", NodeID: 18, DataType: "Double", Unit: "°C", Description: "Product temperature"},
		machine.TagDefinition{Name: "pump_speed_rpm", DisplayName: "Pump Speed (RPM)", NodeID: 19, DataType: "Double", Unit: "RPM", Description: "Peristaltic pump speed"},
		machine.TagDefinition{Name: "dosing_accuracy_pct", DisplayName: "Dosing Accuracy (%)", NodeID: 20, DataType: "Double", Unit: "%", Description: "Dosing accuracy"},
		machine.TagDefinition{Name: "checkweigh_deviation_mg", DisplayName: "Checkweigh Deviation (mg)", NodeID: 21, DataType: "Double", Unit: "mg", Description: "Checkweigh deviation"},
	)
	return defs
}
