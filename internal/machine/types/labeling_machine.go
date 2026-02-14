package types

import (
	"context"
	"time"

	"machine-simulator-2/internal/config"
	"machine-simulator-2/internal/machine"
)

func init() {
	machine.Register("labeling_machine", func(cfg config.MachineInstanceConfig) machine.Machine {
		return NewLabelingMachine(cfg)
	})
}

// LabelingMachineMachine simulates an automated labeling machine.
type LabelingMachineMachine struct {
	*machine.BaseMachine

	labelPosAccMM       float64
	applicatorSpeedLMin float64
	labelTensionN       float64
	printerDPI          float64
	printSpeedMMS       float64
	labelCount          float64
	rejectCount         float64
	ribbonRemainingPct  float64
	labelRemainingPct   float64
	sensorSignalPct     float64
	webSpeedMMin        float64
	regErrorMM          float64
	adhesiveTempC       float64
	rollerPressureN     float64
	cameraVerifyPct     float64
}

func NewLabelingMachine(cfg config.MachineInstanceConfig) *LabelingMachineMachine {
	return &LabelingMachineMachine{
		BaseMachine:         machine.NewBaseMachine(cfg),
		labelPosAccMM:       0.3,
		applicatorSpeedLMin: 200.0,
		labelTensionN:       8.0,
		printerDPI:          300.0,
		printSpeedMMS:       150.0,
		labelCount:          0.0,
		rejectCount:         0.0,
		ribbonRemainingPct:  90.0,
		labelRemainingPct:   85.0,
		sensorSignalPct:     95.0,
		webSpeedMMin:        25.0,
		regErrorMM:          0.1,
		adhesiveTempC:       35.0,
		rollerPressureN:     40.0,
		cameraVerifyPct:     98.0,
	}
}

func (m *LabelingMachineMachine) Start(ctx context.Context) {
	go m.updateTags(ctx)
	m.BaseMachine.Start(ctx)
}

func (m *LabelingMachineMachine) updateTags(ctx context.Context) {
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
				m.labelPosAccMM += (rng.Float64()*2 - 1) * 0.02
				clampF(&m.labelPosAccMM, 0.05, 1.0)
				m.applicatorSpeedLMin += (rng.Float64()*2 - 1) * 2.0
				clampF(&m.applicatorSpeedLMin, 50.0, 400.0)
				m.labelTensionN += (rng.Float64()*2 - 1) * 0.2
				clampF(&m.labelTensionN, 3.0, 15.0)
				m.printerDPI += (rng.Float64()*2 - 1) * 0.1
				clampF(&m.printerDPI, 200.0, 600.0)
				m.printSpeedMMS += (rng.Float64()*2 - 1) * 1.0
				clampF(&m.printSpeedMMS, 80.0, 300.0)
				m.labelCount += rng.Float64() * 2.0
				if rng.Float64() < 0.003 {
					m.rejectCount++
				}
				m.ribbonRemainingPct -= rng.Float64() * 0.01
				clampF(&m.ribbonRemainingPct, 0.0, 100.0)
				m.labelRemainingPct -= rng.Float64() * 0.01
				clampF(&m.labelRemainingPct, 0.0, 100.0)
				m.sensorSignalPct += (rng.Float64()*2 - 1) * 0.2
				clampF(&m.sensorSignalPct, 70.0, 100.0)
				m.webSpeedMMin += (rng.Float64()*2 - 1) * 0.3
				clampF(&m.webSpeedMMin, 10.0, 50.0)
				m.regErrorMM += (rng.Float64()*2 - 1) * 0.01
				clampF(&m.regErrorMM, 0.0, 0.5)
				m.adhesiveTempC += (rng.Float64()*2 - 1) * 0.2
				clampF(&m.adhesiveTempC, 20.0, 50.0)
				m.rollerPressureN += (rng.Float64()*2 - 1) * 0.5
				clampF(&m.rollerPressureN, 15.0, 80.0)
				m.cameraVerifyPct += (rng.Float64()*2 - 1) * 0.2
				clampF(&m.cameraVerifyPct, 85.0, 100.0)
			}
			m.Unlock()
		}
	}
}

func (m *LabelingMachineMachine) GetTags() map[string]interface{} {
	m.RLock()
	defer m.RUnlock()
	tags := machine.BaseTagValues(m.BaseMachine)
	tags["label_position_accuracy_mm"] = m.labelPosAccMM
	tags["applicator_speed_labels_min"] = m.applicatorSpeedLMin
	tags["label_tension_n"] = m.labelTensionN
	tags["printer_dpi"] = m.printerDPI
	tags["print_speed_mm_s"] = m.printSpeedMMS
	tags["label_count"] = m.labelCount
	tags["reject_count"] = m.rejectCount
	tags["ribbon_remaining_pct"] = m.ribbonRemainingPct
	tags["label_remaining_pct"] = m.labelRemainingPct
	tags["sensor_signal_pct"] = m.sensorSignalPct
	tags["web_speed_m_min"] = m.webSpeedMMin
	tags["registration_error_mm"] = m.regErrorMM
	tags["adhesive_temp_c"] = m.adhesiveTempC
	tags["roller_pressure_n"] = m.rollerPressureN
	tags["camera_verify_score_pct"] = m.cameraVerifyPct
	return tags
}

func (m *LabelingMachineMachine) GetTagDefinitions() []machine.TagDefinition {
	defs := machine.BaseTags()
	defs = append(defs,
		machine.TagDefinition{Name: "label_position_accuracy_mm", DisplayName: "Label Position Accuracy (mm)", NodeID: 7, DataType: "Double", Unit: "mm", Description: "Label position accuracy"},
		machine.TagDefinition{Name: "applicator_speed_labels_min", DisplayName: "Applicator Speed (labels/min)", NodeID: 8, DataType: "Double", Unit: "labels/min", Description: "Label applicator speed"},
		machine.TagDefinition{Name: "label_tension_n", DisplayName: "Label Tension (N)", NodeID: 9, DataType: "Double", Unit: "N", Description: "Label web tension"},
		machine.TagDefinition{Name: "printer_dpi", DisplayName: "Printer DPI (DPI)", NodeID: 10, DataType: "Double", Unit: "DPI", Description: "Print resolution"},
		machine.TagDefinition{Name: "print_speed_mm_s", DisplayName: "Print Speed (mm/s)", NodeID: 11, DataType: "Double", Unit: "mm/s", Description: "Print speed"},
		machine.TagDefinition{Name: "label_count", DisplayName: "Label Count", NodeID: 12, DataType: "Double", Unit: "", Description: "Labels applied"},
		machine.TagDefinition{Name: "reject_count", DisplayName: "Reject Count", NodeID: 13, DataType: "Double", Unit: "", Description: "Rejected labels"},
		machine.TagDefinition{Name: "ribbon_remaining_pct", DisplayName: "Ribbon Remaining (%)", NodeID: 14, DataType: "Double", Unit: "%", Description: "Ribbon remaining"},
		machine.TagDefinition{Name: "label_remaining_pct", DisplayName: "Label Remaining (%)", NodeID: 15, DataType: "Double", Unit: "%", Description: "Label stock remaining"},
		machine.TagDefinition{Name: "sensor_signal_pct", DisplayName: "Sensor Signal (%)", NodeID: 16, DataType: "Double", Unit: "%", Description: "Label sensor signal strength"},
		machine.TagDefinition{Name: "web_speed_m_min", DisplayName: "Web Speed (m/min)", NodeID: 17, DataType: "Double", Unit: "m/min", Description: "Web speed"},
		machine.TagDefinition{Name: "registration_error_mm", DisplayName: "Registration Error (mm)", NodeID: 18, DataType: "Double", Unit: "mm", Description: "Registration error"},
		machine.TagDefinition{Name: "adhesive_temp_c", DisplayName: "Adhesive Temp (°C)", NodeID: 19, DataType: "Double", Unit: "°C", Description: "Adhesive temperature"},
		machine.TagDefinition{Name: "roller_pressure_n", DisplayName: "Roller Pressure (N)", NodeID: 20, DataType: "Double", Unit: "N", Description: "Applicator roller pressure"},
		machine.TagDefinition{Name: "camera_verify_score_pct", DisplayName: "Camera Verify Score (%)", NodeID: 21, DataType: "Double", Unit: "%", Description: "Camera verification score"},
	)
	return defs
}
