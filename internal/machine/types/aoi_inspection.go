package types

import (
	"context"
	"time"

	"machine-simulator-2/internal/config"
	"machine-simulator-2/internal/machine"
)

func init() {
	machine.Register("aoi_inspection", func(cfg config.MachineInstanceConfig) machine.Machine {
		return NewAOIInspection(cfg)
	})
}

// AOIInspectionMachine simulates an Automated Optical Inspection system.
type AOIInspectionMachine struct {
	*machine.BaseMachine

	inspectionSpeedCM2S   float64
	cameraResolutionMP    float64
	lightingIntensityLux  float64
	defectCount           float64
	falsePositiveCount    float64
	boardCount            float64
	solderJointCount      float64
	componentCountInsp    float64
	passRatePct           float64
	imageCaptureMS        float64
	processingTimeMS      float64
	fovWidthMM            float64
	fovHeightMM           float64
	magnificationX        float64
	focusScorePct         float64
}

func NewAOIInspection(cfg config.MachineInstanceConfig) *AOIInspectionMachine {
	return &AOIInspectionMachine{
		BaseMachine:          machine.NewBaseMachine(cfg),
		inspectionSpeedCM2S:  15.0,
		cameraResolutionMP:   12.0,
		lightingIntensityLux: 15000.0,
		defectCount:          0.0,
		falsePositiveCount:   0.0,
		boardCount:           0.0,
		solderJointCount:     0.0,
		componentCountInsp:   0.0,
		passRatePct:          97.5,
		imageCaptureMS:       8.0,
		processingTimeMS:     25.0,
		fovWidthMM:           30.0,
		fovHeightMM:          22.0,
		magnificationX:       5.0,
		focusScorePct:        95.0,
	}
}

func (m *AOIInspectionMachine) Start(ctx context.Context) {
	go m.updateTags(ctx)
	m.BaseMachine.Start(ctx)
}

func (m *AOIInspectionMachine) updateTags(ctx context.Context) {
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
				m.inspectionSpeedCM2S += (rng.Float64()*2 - 1) * 0.3
				clampF(&m.inspectionSpeedCM2S, 5.0, 30.0)
				m.cameraResolutionMP += (rng.Float64()*2 - 1) * 0.01
				clampF(&m.cameraResolutionMP, 5.0, 25.0)
				m.lightingIntensityLux += (rng.Float64()*2 - 1) * 50.0
				clampF(&m.lightingIntensityLux, 5000.0, 30000.0)
				if rng.Float64() < 0.01 {
					m.defectCount++
				}
				if rng.Float64() < 0.005 {
					m.falsePositiveCount++
				}
				m.boardCount += rng.Float64() * 0.3
				m.solderJointCount += rng.Float64() * 50.0
				m.componentCountInsp += rng.Float64() * 10.0
				m.passRatePct += (rng.Float64()*2 - 1) * 0.1
				clampF(&m.passRatePct, 85.0, 100.0)
				m.imageCaptureMS += (rng.Float64()*2 - 1) * 0.2
				clampF(&m.imageCaptureMS, 2.0, 20.0)
				m.processingTimeMS += (rng.Float64()*2 - 1) * 0.5
				clampF(&m.processingTimeMS, 10.0, 50.0)
				m.fovWidthMM += (rng.Float64()*2 - 1) * 0.1
				clampF(&m.fovWidthMM, 10.0, 50.0)
				m.fovHeightMM += (rng.Float64()*2 - 1) * 0.1
				clampF(&m.fovHeightMM, 8.0, 40.0)
				m.magnificationX += (rng.Float64()*2 - 1) * 0.05
				clampF(&m.magnificationX, 2.0, 15.0)
				m.focusScorePct += (rng.Float64()*2 - 1) * 0.2
				clampF(&m.focusScorePct, 80.0, 100.0)
			}
			m.Unlock()
		}
	}
}

func (m *AOIInspectionMachine) GetTags() map[string]interface{} {
	m.RLock()
	defer m.RUnlock()
	tags := machine.BaseTagValues(m.BaseMachine)
	tags["inspection_speed_cm2_s"] = m.inspectionSpeedCM2S
	tags["camera_resolution_mp"] = m.cameraResolutionMP
	tags["lighting_intensity_lux"] = m.lightingIntensityLux
	tags["defect_count"] = m.defectCount
	tags["false_positive_count"] = m.falsePositiveCount
	tags["board_count"] = m.boardCount
	tags["solder_joint_count"] = m.solderJointCount
	tags["component_count_inspected"] = m.componentCountInsp
	tags["pass_rate_pct"] = m.passRatePct
	tags["image_capture_ms"] = m.imageCaptureMS
	tags["processing_time_ms"] = m.processingTimeMS
	tags["fov_width_mm"] = m.fovWidthMM
	tags["fov_height_mm"] = m.fovHeightMM
	tags["magnification_x"] = m.magnificationX
	tags["focus_score_pct"] = m.focusScorePct
	return tags
}

func (m *AOIInspectionMachine) GetTagDefinitions() []machine.TagDefinition {
	defs := machine.BaseTags()
	defs = append(defs,
		machine.TagDefinition{Name: "inspection_speed_cm2_s", DisplayName: "Inspection Speed (cm\u00b2/s)", NodeID: 7, DataType: "Double", Unit: "cm²/s", Description: "Inspection speed"},
		machine.TagDefinition{Name: "camera_resolution_mp", DisplayName: "Camera Resolution (MP)", NodeID: 8, DataType: "Double", Unit: "MP", Description: "Camera resolution"},
		machine.TagDefinition{Name: "lighting_intensity_lux", DisplayName: "Lighting Intensity (lux)", NodeID: 9, DataType: "Double", Unit: "lux", Description: "Lighting intensity"},
		machine.TagDefinition{Name: "defect_count", DisplayName: "Defect Count", NodeID: 10, DataType: "Double", Unit: "", Description: "Defects detected"},
		machine.TagDefinition{Name: "false_positive_count", DisplayName: "False Positive Count", NodeID: 11, DataType: "Double", Unit: "", Description: "False positive detections"},
		machine.TagDefinition{Name: "board_count", DisplayName: "Board Count", NodeID: 12, DataType: "Double", Unit: "", Description: "Boards inspected"},
		machine.TagDefinition{Name: "solder_joint_count", DisplayName: "Solder Joint Count", NodeID: 13, DataType: "Double", Unit: "", Description: "Solder joints inspected"},
		machine.TagDefinition{Name: "component_count_inspected", DisplayName: "Components Inspected", NodeID: 14, DataType: "Double", Unit: "", Description: "Components inspected"},
		machine.TagDefinition{Name: "pass_rate_pct", DisplayName: "Pass Rate (%)", NodeID: 15, DataType: "Double", Unit: "%", Description: "Inspection pass rate"},
		machine.TagDefinition{Name: "image_capture_ms", DisplayName: "Image Capture Time (ms)", NodeID: 16, DataType: "Double", Unit: "ms", Description: "Image capture time"},
		machine.TagDefinition{Name: "processing_time_ms", DisplayName: "Processing Time (ms)", NodeID: 17, DataType: "Double", Unit: "ms", Description: "Image processing time"},
		machine.TagDefinition{Name: "fov_width_mm", DisplayName: "FOV Width (mm)", NodeID: 18, DataType: "Double", Unit: "mm", Description: "Field of view width"},
		machine.TagDefinition{Name: "fov_height_mm", DisplayName: "FOV Height (mm)", NodeID: 19, DataType: "Double", Unit: "mm", Description: "Field of view height"},
		machine.TagDefinition{Name: "magnification_x", DisplayName: "Magnification (x)", NodeID: 20, DataType: "Double", Unit: "x", Description: "Optical magnification"},
		machine.TagDefinition{Name: "focus_score_pct", DisplayName: "Focus Score (%)", NodeID: 21, DataType: "Double", Unit: "%", Description: "Auto-focus quality score"},
	)
	return defs
}
