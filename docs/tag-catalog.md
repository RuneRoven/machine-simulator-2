# Tag Catalog

## Introduction

Every simulated machine in the machine-simulator exposes a set of **tags** -- named data points that represent sensors, counters, and configuration values. Tags are the primary interface for reading real-time machine data via OPC-UA or Modbus.

Each machine exposes two layers of tags:

1. **Base tags** (NodeID 1--6) -- present on every machine regardless of type. These cover universal concerns such as machine state, cycle counting, and buffer status.
2. **Machine-type tags** (NodeID 7+) -- specific to each of the 27 machine types. These represent the sensors and metrics unique to that class of equipment.
3. **Flow tags** (NodeID 1001--1005) -- present on every machine: whether it waits for a part, and the level and capacity of the buffers before and after it. They use a separate range so that existing NodeIDs never move; in the Modbus register map they come after all other tags.

Tag definitions are authoritative; they are derived directly from the `GetTagDefinitions()` functions in the Go source files under `internal/machine/types/`.

---

## OPC-UA Addressing Guide

Each tag is assigned a numeric **NodeID** that is unique within a single machine. To address a tag via OPC-UA, use the namespace-qualified identifier:

```
ns=1;i=<NodeID>
```

For example, to read the `state` tag (NodeID 1) of a machine:

```
ns=1;i=1
```

To read the `press_force_kn` tag (NodeID 7) of a metal forming machine:

```
ns=1;i=7
```

Each machine instance runs its own OPC-UA server (or is a node within a shared server), so the same NodeID space applies independently per machine. NodeIDs 1--6 are always the base tags; NodeIDs 7+ are type-specific.

---

## Flow Tags

| NodeID | Tag | Type | Description |
|---|---|---|---|
| 1001 | `starved` | Boolean | Waiting for a part: empty in-buffer, or no release at stage 1 (no running job, or target reached) |
| 1002 | `in_buffer_level` | Int32 | Parts in the buffer feeding this machine (0 at stage 1) |
| 1003 | `in_buffer_capacity` | Int32 | Capacity of that buffer (0 at stage 1) |
| 1004 | `out_buffer_level` | Int32 | Parts in the buffer after this machine (0 at the last stage) |
| 1005 | `out_buffer_capacity` | Int32 | Capacity of that buffer (0 at the last stage) |

---

## Base Tags

These 6 tags are present on **every** machine, regardless of machine type.

| Name | NodeID | DisplayName | DataType | Unit | Description |
|------|--------|-------------|----------|------|-------------|
| `state` | 1 | State | String | | Machine state (IDLE, RUNNING, STOPPED, FAULT, MAINTENANCE) |
| `cycle_count` | 2 | Cycle Count | Int64 | | Total cycles completed |
| `good_count` | 3 | Good Count | Int64 | | Good parts produced |
| `scrap_count` | 4 | Scrap Count | Int64 | | Scrap parts produced |
| `cycle_time_ms` | 5 | Cycle Time (ms) | Int32 | ms | Current cycle time setting |
| `blocked_by_buffer` | 6 | Blocked By Buffer | Boolean | | Whether machine is blocked by a full downstream buffer |

---

## Machine Type Tags

Below are the type-specific tags for each of the 27 machine types, grouped by industry domain. Every machine also includes the 6 base tags listed above (NodeIDs 1--6).

---

### Automotive

#### metal_forming

Simulates a hydraulic metal forming press.

Source: `internal/machine/types/metal_forming.go`

| Name | NodeID | DisplayName | DataType | Unit | Description |
|------|--------|-------------|----------|------|-------------|
| `press_force_kn` | 7 | Press Force (kN) | Double | kN | Press force |
| `hydraulic_pressure_bar` | 8 | Hydraulic Pressure (bar) | Double | bar | Hydraulic system pressure |
| `die_temperature_c` | 9 | Die Temperature (°C) | Double | °C | Die temperature |
| `stroke_position_mm` | 10 | Stroke Position (mm) | Double | mm | Ram stroke position |
| `tool_wear_pct` | 11 | Tool Wear (%) | Double | % | Tool wear percentage |
| `ram_speed_mm_s` | 12 | Ram Speed (mm/s) | Double | mm/s | Ram speed |
| `blank_holder_force_kn` | 13 | Blank Holder Force (kN) | Double | kN | Blank holder force |
| `material_thickness_mm` | 14 | Material Thickness (mm) | Double | mm | Material thickness |
| `tonnage_pct` | 15 | Tonnage Utilization (%) | Double | % | Tonnage utilization |
| `lubrication_level_pct` | 16 | Lubrication Level (%) | Double | % | Lubrication level |
| `vibration_mm_s` | 17 | Vibration Level (mm/s) | Double | mm/s | Vibration level |
| `noise_level_db` | 18 | Noise Level (dB) | Double | dB | Noise level |
| `motor_current_a` | 19 | Motor Current (A) | Double | A | Motor current draw |
| `hydraulic_oil_temp_c` | 20 | Hydraulic Oil Temp (°C) | Double | °C | Hydraulic oil temperature |
| `cycle_energy_kwh` | 21 | Cycle Energy (kWh) | Double | kWh | Energy consumed per cycle |

#### spot_welder

Simulates a resistance spot welding machine.

Source: `internal/machine/types/spot_welder.go`

| Name | NodeID | DisplayName | DataType | Unit | Description |
|------|--------|-------------|----------|------|-------------|
| `weld_current_ka` | 7 | Welding Current (kA) | Double | kA | Welding current |
| `electrode_force_kn` | 8 | Electrode Force (kN) | Double | kN | Electrode force |
| `weld_time_ms` | 9 | Weld Duration (ms) | Double | ms | Weld duration |
| `nugget_diameter_mm` | 10 | Nugget Diameter (mm) | Double | mm | Weld nugget diameter |
| `tip_wear_pct` | 11 | Tip Wear (%) | Double | % | Electrode tip wear |
| `squeeze_time_ms` | 12 | Squeeze Time (ms) | Double | ms | Squeeze time before weld |
| `hold_time_ms` | 13 | Hold Time (ms) | Double | ms | Hold time after weld |
| `electrode_temp_c` | 14 | Electrode Temp (°C) | Double | °C | Electrode temperature |
| `water_flow_l_min` | 15 | Water Flow Rate (L/min) | Double | L/min | Cooling water flow rate |
| `resistance_mohm` | 16 | Contact Resistance (mΩ) | Double | mohm | Contact resistance |
| `expulsion_count` | 17 | Expulsion Count | Double | | Expulsion event count |
| `weld_power_kw` | 18 | Welding Power (kW) | Double | kW | Welding power |
| `cap_displacement_mm` | 19 | Cap Displacement (mm) | Double | mm | Cap tip displacement |
| `shunt_factor` | 20 | Shunt Factor | Double | | Current shunt factor |
| `sheet_gap_mm` | 21 | Sheet Gap (mm) | Double | mm | Gap between sheets |

#### robot_welder

Simulates a 6-axis robotic arc welding cell.

Source: `internal/machine/types/robot_welder.go`

| Name | NodeID | DisplayName | DataType | Unit | Description |
|------|--------|-------------|----------|------|-------------|
| `joint1_temp_c` | 7 | Joint 1 Temp (°C) | Double | °C | Joint 1 temperature |
| `joint2_temp_c` | 8 | Joint 2 Temp (°C) | Double | °C | Joint 2 temperature |
| `joint3_temp_c` | 9 | Joint 3 Temp (°C) | Double | °C | Joint 3 temperature |
| `joint4_temp_c` | 10 | Joint 4 Temp (°C) | Double | °C | Joint 4 temperature |
| `joint5_temp_c` | 11 | Joint 5 Temp (°C) | Double | °C | Joint 5 temperature |
| `joint6_temp_c` | 12 | Joint 6 Temp (°C) | Double | °C | Joint 6 temperature |
| `arc_voltage_v` | 13 | Arc Voltage (V) | Double | V | Arc voltage |
| `arc_current_a` | 14 | Arc Current (A) | Double | A | Arc current |
| `wire_feed_m_min` | 15 | Wire Feed Speed (m/min) | Double | m/min | Wire feed speed |
| `gas_flow_l_min` | 16 | Gas Flow Rate (L/min) | Double | L/min | Shielding gas flow rate |
| `travel_speed_mm_s` | 17 | Travel Speed (mm/s) | Double | mm/s | Torch travel speed |
| `weld_penetration_mm` | 18 | Weld Penetration (mm) | Double | mm | Weld penetration depth |
| `torch_angle_deg` | 19 | Torch Angle (°) | Double | deg | Torch angle |
| `seam_tracking_offset_mm` | 20 | Seam Tracking Offset (mm) | Double | mm | Seam tracking offset |
| `spatter_count` | 21 | Spatter Count | Double | | Weld spatter event count |

#### robot_pick_place

Simulates a 6-axis pick-and-place robot.

Source: `internal/machine/types/robot_pick_place.go`

| Name | NodeID | DisplayName | DataType | Unit | Description |
|------|--------|-------------|----------|------|-------------|
| `joint1_angle` | 7 | Joint 1 Angle (°) | Double | deg | Joint 1 angle |
| `joint2_angle` | 8 | Joint 2 Angle (°) | Double | deg | Joint 2 angle |
| `joint3_angle` | 9 | Joint 3 Angle (°) | Double | deg | Joint 3 angle |
| `joint4_angle` | 10 | Joint 4 Angle (°) | Double | deg | Joint 4 angle |
| `joint5_angle` | 11 | Joint 5 Angle (°) | Double | deg | Joint 5 angle |
| `joint6_angle` | 12 | Joint 6 Angle (°) | Double | deg | Joint 6 angle |
| `tcp_x` | 13 | TCP X Position (mm) | Double | mm | Tool center point X position |
| `tcp_y` | 14 | TCP Y Position (mm) | Double | mm | Tool center point Y position |
| `tcp_z` | 15 | TCP Z Position (mm) | Double | mm | Tool center point Z position |
| `gripper_force_n` | 16 | Gripper Force (N) | Double | N | Gripper force |
| `vacuum_pressure_kpa` | 17 | Vacuum Pressure (kPa) | Double | kPa | Vacuum pressure for suction gripper |
| `pick_accuracy_um` | 18 | Pick Accuracy (µm) | Double | um | Pick position accuracy |
| `place_accuracy_um` | 19 | Place Accuracy (µm) | Double | um | Place position accuracy |
| `cycle_speed_pct` | 20 | Cycle Speed (%) | Double | % | Cycle speed percentage |
| `payload_kg` | 21 | Payload Weight (kg) | Double | kg | Current payload weight |
| `acceleration_m_s2` | 22 | Acceleration (m/s²) | Double | m/s2 | Acceleration |

#### painting_booth

Simulates an automotive paint spray booth.

Source: `internal/machine/types/painting_booth.go`

| Name | NodeID | DisplayName | DataType | Unit | Description |
|------|--------|-------------|----------|------|-------------|
| `booth_temp_c` | 7 | Booth Temperature (°C) | Double | °C | Booth temperature |
| `booth_humidity_pct` | 8 | Booth Humidity (%) | Double | % | Booth humidity |
| `paint_flow_ml_min` | 9 | Paint Flow Rate (mL/min) | Double | mL/min | Paint flow rate |
| `atomization_pressure_bar` | 10 | Atomization Pressure (bar) | Double | bar | Atomization air pressure |
| `film_thickness_um` | 11 | Film Thickness (µm) | Double | um | Paint film thickness |
| `overspray_pct` | 12 | Overspray (%) | Double | % | Overspray percentage |
| `fan_speed_rpm` | 13 | Fan Speed (RPM) | Double | RPM | Exhaust fan speed |
| `filter_pressure_drop_pa` | 14 | Filter Pressure Drop (Pa) | Double | Pa | Filter pressure drop |
| `solvent_concentration_ppm` | 15 | Solvent Concentration (ppm) | Double | ppm | Solvent vapor concentration |
| `cure_temp_c` | 16 | Cure Oven Temp (°C) | Double | °C | Cure oven temperature |
| `conveyor_speed_m_min` | 17 | Conveyor Speed (m/min) | Double | m/min | Conveyor speed |
| `paint_viscosity_cp` | 18 | Paint Viscosity (cP) | Double | cP | Paint viscosity |
| `color_delta_e` | 19 | Color Deviation (ΔE) | Double | dE | Color deviation |
| `booth_airflow_m3_h` | 20 | Booth Airflow (m³/h) | Double | m3/h | Booth airflow volume |
| `paint_waste_pct` | 21 | Paint Waste (%) | Double | % | Paint waste percentage |

---

### Electronics

#### smt_placement

Simulates a surface-mount technology pick-and-place machine.

Source: `internal/machine/types/smt_placement.go`

| Name | NodeID | DisplayName | DataType | Unit | Description |
|------|--------|-------------|----------|------|-------------|
| `placement_rate_cph` | 7 | Placement Rate (cph) | Double | cph | Component placement rate |
| `nozzle_vacuum_kpa` | 8 | Nozzle Vacuum (kPa) | Double | kPa | Nozzle vacuum pressure |
| `feeder_status_pct` | 9 | Feeder Availability (%) | Double | % | Feeder availability |
| `placement_accuracy_um` | 10 | Placement Accuracy (µm) | Double | um | Placement accuracy |
| `head_x_mm` | 11 | Head X Position (mm) | Double | mm | Head X position |
| `head_y_mm` | 12 | Head Y Position (mm) | Double | mm | Head Y position |
| `vision_score_pct` | 13 | Vision Score (%) | Double | % | Vision system alignment score |
| `component_count` | 14 | Component Count | Double | | Components placed |
| `reject_count` | 15 | Reject Count | Double | | Rejected placements |
| `nozzle_temp_c` | 16 | Nozzle Temp (°C) | Double | °C | Nozzle temperature |
| `board_temp_c` | 17 | Board Temp (°C) | Double | °C | Board temperature |
| `tape_tension_n` | 18 | Tape Tension (N) | Double | N | Tape feeder tension |
| `conveyor_width_mm` | 19 | Conveyor Width (mm) | Double | mm | Conveyor width setting |
| `rotation_accuracy_deg` | 20 | Rotation Accuracy (°) | Double | deg | Component rotation accuracy |
| `pick_failure_count` | 21 | Pick Failure Count | Double | | Pick failure count |

#### reflow_oven

Simulates a multi-zone reflow soldering oven.

Source: `internal/machine/types/reflow_oven.go`

| Name | NodeID | DisplayName | DataType | Unit | Description |
|------|--------|-------------|----------|------|-------------|
| `zone1_temp_c` | 7 | Preheat Zone Temp (°C) | Double | °C | Preheat zone temperature |
| `zone2_temp_c` | 8 | Soak Zone Temp (°C) | Double | °C | Soak zone temperature |
| `zone3_temp_c` | 9 | Reflow Zone Temp (°C) | Double | °C | Reflow zone temperature |
| `zone4_temp_c` | 10 | Cooling Zone Temp (°C) | Double | °C | Cooling zone temperature |
| `conveyor_speed_cm_min` | 11 | Conveyor Speed (cm/min) | Double | cm/min | Conveyor speed |
| `oxygen_level_ppm` | 12 | Oxygen Level (ppm) | Double | ppm | Oxygen level in oven |
| `nitrogen_flow_l_min` | 13 | Nitrogen Flow (L/min) | Double | L/min | Nitrogen flow rate |
| `peak_temp_c` | 14 | Peak Reflow Temp (°C) | Double | °C | Peak reflow temperature |
| `time_above_liquidus_s` | 15 | Time Above Liquidus (s) | Double | s | Time above solder liquidus |
| `heating_rate_c_s` | 16 | Heating Rate (°C/s) | Double | °C/s | Heating rate |
| `cooling_rate_c_s` | 17 | Cooling Rate (°C/s) | Double | °C/s | Cooling rate |
| `board_count` | 18 | Board Count | Double | | Boards processed |
| `power_consumption_kw` | 19 | Power Consumption (kW) | Double | kW | Power consumption |
| `exhaust_temp_c` | 20 | Exhaust Temp (°C) | Double | °C | Exhaust temperature |
| `belt_width_mm` | 21 | Belt Width (mm) | Double | mm | Belt width |

#### wave_solder

Simulates a wave soldering machine.

Source: `internal/machine/types/wave_solder.go`

| Name | NodeID | DisplayName | DataType | Unit | Description |
|------|--------|-------------|----------|------|-------------|
| `solder_pot_temp_c` | 7 | Solder Pot Temp (°C) | Double | °C | Solder pot temperature |
| `wave_height_mm` | 8 | Wave Height (mm) | Double | mm | Solder wave height |
| `flux_density_ml_m2` | 9 | Flux Density (mL/m²) | Double | mL/m2 | Flux density |
| `conveyor_speed_m_min` | 10 | Conveyor Speed (m/min) | Double | m/min | Conveyor speed |
| `preheat_top_temp_c` | 11 | Top Preheat Temp (°C) | Double | °C | Top preheat temperature |
| `preheat_bottom_temp_c` | 12 | Bottom Preheat Temp (°C) | Double | °C | Bottom preheat temperature |
| `solder_purity_pct` | 13 | Solder Purity (%) | Double | % | Solder purity level |
| `dross_level_pct` | 14 | Dross Level (%) | Double | % | Dross accumulation level |
| `finger_clearance_mm` | 15 | Finger Clearance (mm) | Double | mm | Solder finger clearance |
| `contact_time_s` | 16 | Contact Time (s) | Double | s | Board-wave contact time |
| `board_angle_deg` | 17 | Board Angle (°) | Double | deg | Board conveyor angle |
| `nitrogen_flow_l_min` | 18 | Nitrogen Flow (L/min) | Double | L/min | Nitrogen flow rate |
| `pot_level_mm` | 19 | Pot Level (mm) | Double | mm | Solder pot level |
| `bridge_defect_count` | 20 | Bridge Defect Count | Double | | Solder bridge defect count |
| `icicle_count` | 21 | Icicle Defect Count | Double | | Solder icicle defect count |

#### aoi_inspection

Simulates an Automated Optical Inspection system.

Source: `internal/machine/types/aoi_inspection.go`

| Name | NodeID | DisplayName | DataType | Unit | Description |
|------|--------|-------------|----------|------|-------------|
| `inspection_speed_cm2_s` | 7 | Inspection Speed (cm²/s) | Double | cm2/s | Inspection speed |
| `camera_resolution_mp` | 8 | Camera Resolution (MP) | Double | MP | Camera resolution |
| `lighting_intensity_lux` | 9 | Lighting Intensity (lux) | Double | lux | Lighting intensity |
| `defect_count` | 10 | Defect Count | Double | | Defects detected |
| `false_positive_count` | 11 | False Positive Count | Double | | False positive detections |
| `board_count` | 12 | Board Count | Double | | Boards inspected |
| `solder_joint_count` | 13 | Solder Joint Count | Double | | Solder joints inspected |
| `component_count_inspected` | 14 | Components Inspected | Double | | Components inspected |
| `pass_rate_pct` | 15 | Pass Rate (%) | Double | % | Inspection pass rate |
| `image_capture_ms` | 16 | Image Capture Time (ms) | Double | ms | Image capture time |
| `processing_time_ms` | 17 | Processing Time (ms) | Double | ms | Image processing time |
| `fov_width_mm` | 18 | FOV Width (mm) | Double | mm | Field of view width |
| `fov_height_mm` | 19 | FOV Height (mm) | Double | mm | Field of view height |
| `magnification_x` | 20 | Magnification (x) | Double | x | Optical magnification |
| `focus_score_pct` | 21 | Focus Score (%) | Double | % | Auto-focus quality score |

---

### Food & Beverage

#### filling_machine

Simulates a liquid filling machine for food and beverage.

Source: `internal/machine/types/filling_machine.go`

| Name | NodeID | DisplayName | DataType | Unit | Description |
|------|--------|-------------|----------|------|-------------|
| `fill_volume_ml` | 7 | Fill Volume (mL) | Double | mL | Fill volume per container |
| `fill_rate_ml_s` | 8 | Fill Rate (mL/s) | Double | mL/s | Fill rate |
| `nozzle_pressure_bar` | 9 | Nozzle Pressure (bar) | Double | bar | Nozzle pressure |
| `product_temp_c` | 10 | Product Temp (°C) | Double | °C | Product temperature |
| `tank_level_pct` | 11 | Tank Level (%) | Double | % | Supply tank level |
| `flow_rate_l_min` | 12 | Flow Rate (L/min) | Double | L/min | Product flow rate |
| `fill_accuracy_pct` | 13 | Fill Accuracy (%) | Double | % | Fill accuracy |
| `drip_count` | 14 | Drip Count | Double | | Nozzle drip events |
| `valve_cycle_count` | 15 | Valve Cycle Count | Double | | Fill valve cycles |
| `cip_status` | 16 | CIP Status | Double | | Clean-in-place status (0=idle, 1=active) |
| `product_viscosity_cp` | 17 | Product Viscosity (cP) | Double | cP | Product viscosity |
| `line_pressure_bar` | 18 | Line Pressure (bar) | Double | bar | Line pressure |
| `fill_weight_g` | 19 | Fill Weight (g) | Double | g | Measured fill weight |
| `density_kg_m3` | 20 | Density (kg/m3) | Double | kg/m3 | Product density |
| `foam_level_pct` | 21 | Foam Level (%) | Double | % | Foam level in container |

#### sealing_machine

Simulates a heat sealing machine for packaging.

Source: `internal/machine/types/sealing_machine.go`

| Name | NodeID | DisplayName | DataType | Unit | Description |
|------|--------|-------------|----------|------|-------------|
| `seal_temp_c` | 7 | Seal Temp (°C) | Double | °C | Seal bar temperature |
| `seal_pressure_bar` | 8 | Seal Pressure (bar) | Double | bar | Seal pressure |
| `seal_time_ms` | 9 | Seal Time (ms) | Double | ms | Seal dwell time |
| `conveyor_speed_m_min` | 10 | Conveyor Speed (m/min) | Double | m/min | Conveyor speed |
| `jaw_gap_mm` | 11 | Jaw Gap (mm) | Double | mm | Jaw gap |
| `seal_strength_n` | 12 | Seal Strength (N) | Double | N | Seal strength |
| `peel_force_n` | 13 | Peel Force (N) | Double | N | Peel force |
| `seal_width_mm` | 14 | Seal Width (mm) | Double | mm | Seal width |
| `cooling_water_temp_c` | 15 | Cooling Water Temp (°C) | Double | °C | Cooling water temperature |
| `film_tension_n` | 16 | Film Tension (N) | Double | N | Film tension |
| `reject_count` | 17 | Reject Count | Double | | Rejected seals |
| `vacuum_level_mbar` | 18 | Vacuum Level (mbar) | Double | mbar | Vacuum level for MAP |
| `gas_flush_flow_l_min` | 19 | Gas Flush Flow (L/min) | Double | L/min | Gas flush flow rate |
| `leak_rate_mbar_l_s` | 20 | Leak Rate (mbar·L/s) | Double | mbar*L/s | Leak rate |
| `heater_power_pct` | 21 | Heater Power (%) | Double | % | Heater power percentage |

#### labeling_machine

Simulates an automated labeling machine.

Source: `internal/machine/types/labeling_machine.go`

| Name | NodeID | DisplayName | DataType | Unit | Description |
|------|--------|-------------|----------|------|-------------|
| `label_position_accuracy_mm` | 7 | Label Position Accuracy (mm) | Double | mm | Label position accuracy |
| `applicator_speed_labels_min` | 8 | Applicator Speed (labels/min) | Double | labels/min | Label applicator speed |
| `label_tension_n` | 9 | Label Tension (N) | Double | N | Label web tension |
| `printer_dpi` | 10 | Printer DPI (DPI) | Double | DPI | Print resolution |
| `print_speed_mm_s` | 11 | Print Speed (mm/s) | Double | mm/s | Print speed |
| `label_count` | 12 | Label Count | Double | | Labels applied |
| `reject_count` | 13 | Reject Count | Double | | Rejected labels |
| `ribbon_remaining_pct` | 14 | Ribbon Remaining (%) | Double | % | Ribbon remaining |
| `label_remaining_pct` | 15 | Label Remaining (%) | Double | % | Label stock remaining |
| `sensor_signal_pct` | 16 | Sensor Signal (%) | Double | % | Label sensor signal strength |
| `web_speed_m_min` | 17 | Web Speed (m/min) | Double | m/min | Web speed |
| `registration_error_mm` | 18 | Registration Error (mm) | Double | mm | Registration error |
| `adhesive_temp_c` | 19 | Adhesive Temp (°C) | Double | °C | Adhesive temperature |
| `roller_pressure_n` | 20 | Roller Pressure (N) | Double | N | Applicator roller pressure |
| `camera_verify_score_pct` | 21 | Camera Verify Score (%) | Double | % | Camera verification score |

---

### Pharma

#### reactor_vessel

Simulates a pharmaceutical reactor vessel.

Source: `internal/machine/types/reactor_vessel.go`

| Name | NodeID | DisplayName | DataType | Unit | Description |
|------|--------|-------------|----------|------|-------------|
| `vessel_temp_c` | 7 | Vessel Temp (°C) | Double | °C | Vessel temperature |
| `vessel_pressure_bar` | 8 | Vessel Pressure (bar) | Double | bar | Vessel pressure |
| `agitator_speed_rpm` | 9 | Agitator Speed (RPM) | Double | RPM | Agitator speed |
| `ph_level` | 10 | pH Level (pH) | Double | pH | pH level |
| `batch_volume_l` | 11 | Batch Volume (L) | Double | L | Batch volume |
| `dissolved_oxygen_ppm` | 12 | Dissolved Oxygen (ppm) | Double | ppm | Dissolved oxygen |
| `conductivity_us_cm` | 13 | Conductivity (uS/cm) | Double | uS/cm | Conductivity |
| `turbidity_ntu` | 14 | Turbidity (NTU) | Double | NTU | Turbidity |
| `jacket_temp_c` | 15 | Jacket Temp (°C) | Double | °C | Jacket temperature |
| `feed_rate_ml_min` | 16 | Feed Rate (mL/min) | Double | mL/min | Feed rate |
| `off_gas_flow_l_min` | 17 | Off-Gas Flow (L/min) | Double | L/min | Off-gas flow rate |
| `torque_nm` | 18 | Torque (N·m) | Double | N·m | Agitator torque |
| `impeller_tip_speed_m_s` | 19 | Impeller Tip Speed (m/s) | Double | m/s | Impeller tip speed |
| `redox_potential_mv` | 20 | Redox Potential (mV) | Double | mV | Redox potential |
| `foam_level_pct` | 21 | Foam Level (%) | Double | % | Foam level |

#### pharma_filler

Simulates a pharmaceutical aseptic vial filler.

Source: `internal/machine/types/pharma_filler.go`

| Name | NodeID | DisplayName | DataType | Unit | Description |
|------|--------|-------------|----------|------|-------------|
| `fill_volume_ml` | 7 | Fill Volume (mL) | Double | mL | Fill volume per vial |
| `fill_accuracy_pct` | 8 | Fill Accuracy (%) | Double | % | Fill accuracy |
| `particulate_count` | 9 | Particulate Count | Double | | Particulate count per sample |
| `clean_room_pressure_pa` | 10 | Clean Room Pressure (Pa) | Double | Pa | Clean room differential pressure |
| `laminar_flow_velocity_m_s` | 11 | Laminar Flow Velocity (m/s) | Double | m/s | Laminar airflow velocity |
| `stopper_insertion_force_n` | 12 | Stopper Force (N) | Double | N | Stopper insertion force |
| `vial_count` | 13 | Vial Count | Double | | Vials filled |
| `reject_count` | 14 | Reject Count | Double | | Rejected vials |
| `fill_weight_mg` | 15 | Fill Weight (mg) | Double | mg | Measured fill weight |
| `tare_weight_mg` | 16 | Tare Weight (mg) | Double | mg | Tare weight of empty vial |
| `nitrogen_overlay_flow_l_min` | 17 | Nitrogen Flow (L/min) | Double | L/min | Nitrogen overlay flow |
| `product_temp_c` | 18 | Product Temp (°C) | Double | °C | Product temperature |
| `pump_speed_rpm` | 19 | Pump Speed (RPM) | Double | RPM | Peristaltic pump speed |
| `dosing_accuracy_pct` | 20 | Dosing Accuracy (%) | Double | % | Dosing accuracy |
| `checkweigh_deviation_mg` | 21 | Checkweigh Deviation (mg) | Double | mg | Checkweigh deviation |

#### capping_machine

Simulates a pharmaceutical capping machine.

Source: `internal/machine/types/capping_machine.go`

| Name | NodeID | DisplayName | DataType | Unit | Description |
|------|--------|-------------|----------|------|-------------|
| `torque_nm` | 7 | Torque (N·m) | Double | N·m | Capping torque |
| `cap_position_accuracy_mm` | 8 | Cap Position Accuracy (mm) | Double | mm | Cap position accuracy |
| `reject_rate_pct` | 9 | Reject Rate (%) | Double | % | Reject rate |
| `feed_rate_caps_min` | 10 | Feed Rate (caps/min) | Double | caps/min | Cap feed rate |
| `cap_presence_pct` | 11 | Cap Presence (%) | Double | % | Cap presence detection |
| `seal_integrity_pct` | 12 | Seal Integrity (%) | Double | % | Seal integrity |
| `chuck_speed_rpm` | 13 | Chuck Speed (RPM) | Double | RPM | Chuck rotation speed |
| `cam_track_force_n` | 14 | Cam Track Force (N) | Double | N | Cam track force |
| `cap_height_mm` | 15 | Cap Height (mm) | Double | mm | Cap height |
| `skirt_diameter_mm` | 16 | Skirt Diameter (mm) | Double | mm | Skirt diameter |
| `liner_compression_pct` | 17 | Liner Compression (%) | Double | % | Liner compression |
| `elevator_level_pct` | 18 | Elevator Level (%) | Double | % | Cap elevator level |
| `servo_current_a` | 19 | Servo Current (A) | Double | A | Servo motor current |
| `vibration_mm_s` | 20 | Vibration (mm/s) | Double | mm/s | Vibration level |
| `cap_orientation_score_pct` | 21 | Cap Orientation Score (%) | Double | % | Cap orientation score |

---

### Window

#### profile_cutter

Simulates a PVC/aluminum profile cutting machine for windows.

Source: `internal/machine/types/profile_cutter.go`

| Name | NodeID | DisplayName | DataType | Unit | Description |
|------|--------|-------------|----------|------|-------------|
| `blade_speed_rpm` | 7 | Blade Speed (RPM) | Double | RPM | Blade rotation speed |
| `cut_length_mm` | 8 | Cut Length (mm) | Double | mm | Cut length |
| `feed_rate_mm_s` | 9 | Feed Rate (mm/s) | Double | mm/s | Feed rate |
| `blade_temp_c` | 10 | Blade Temp (°C) | Double | °C | Blade temperature |
| `angle_accuracy_deg` | 11 | Angle Accuracy (°) | Double | deg | Cut angle accuracy |
| `cut_depth_mm` | 12 | Cut Depth (mm) | Double | mm | Cut depth |
| `clamp_pressure_bar` | 13 | Clamp Pressure (bar) | Double | bar | Clamp pressure |
| `material_hardness_hrc` | 14 | Material Hardness (HRC) | Double | HRC | Material hardness |
| `chip_extraction_m3_h` | 15 | Chip Extraction (m³/h) | Double | m3/h | Chip extraction airflow |
| `blade_wear_pct` | 16 | Blade Wear (%) | Double | % | Blade wear percentage |
| `power_consumption_kw` | 17 | Power Consumption (kW) | Double | kW | Power consumption |
| `vibration_mm_s` | 18 | Vibration (mm/s) | Double | mm/s | Vibration level |
| `coolant_flow_l_min` | 19 | Coolant Flow (L/min) | Double | L/min | Coolant flow rate |
| `surface_roughness_um` | 20 | Surface Roughness (µm) | Double | um | Surface roughness |
| `noise_level_db` | 21 | Noise Level (dB) | Double | dB | Noise level |

#### corner_welder

Simulates a PVC window corner welding machine.

Source: `internal/machine/types/corner_welder.go`

| Name | NodeID | DisplayName | DataType | Unit | Description |
|------|--------|-------------|----------|------|-------------|
| `weld_temp_c` | 7 | Weld Temp (°C) | Double | °C | Weld temperature |
| `weld_time_s` | 8 | Weld Time (s) | Double | s | Weld duration |
| `clamp_force_kn` | 9 | Clamp Force (kN) | Double | kN | Clamp force |
| `joint_strength_n` | 10 | Joint Strength (N) | Double | N | Weld joint strength |
| `alignment_accuracy_mm` | 11 | Alignment Accuracy (mm) | Double | mm | Profile alignment accuracy |
| `heating_plate_temp_c` | 12 | Heating Plate Temp (°C) | Double | °C | Heating plate temperature |
| `melt_depth_mm` | 13 | Melt Depth (mm) | Double | mm | Melt depth |
| `joining_pressure_bar` | 14 | Joining Pressure (bar) | Double | bar | Joining pressure |
| `cooling_time_s` | 15 | Cooling Time (s) | Double | s | Cooling time |
| `bead_height_mm` | 16 | Bead Height (mm) | Double | mm | Weld bead height |
| `weld_flash_mm` | 17 | Weld Flash (mm) | Double | mm | Weld flash height |
| `corner_angle_deg` | 18 | Corner Angle (°) | Double | deg | Corner angle |
| `servo_position_mm` | 19 | Servo Position (mm) | Double | mm | Servo position |
| `power_consumption_kw` | 20 | Power Consumption (kW) | Double | kW | Power consumption |
| `cycle_pressure_bar` | 21 | Cycle Pressure (bar) | Double | bar | Cycle pressure |

#### glass_setter

Simulates a window glass setting/glazing machine.

Source: `internal/machine/types/glass_setter.go`

| Name | NodeID | DisplayName | DataType | Unit | Description |
|------|--------|-------------|----------|------|-------------|
| `suction_vacuum_kpa` | 7 | Suction Vacuum (kPa) | Double | kPa | Suction cup vacuum pressure |
| `glass_thickness_mm` | 8 | Glass Thickness (mm) | Double | mm | Glass thickness |
| `sealant_flow_ml_min` | 9 | Sealant Flow (mL/min) | Double | mL/min | Sealant flow rate |
| `positioning_accuracy_mm` | 10 | Positioning Accuracy (mm) | Double | mm | Glass positioning accuracy |
| `glazing_depth_mm` | 11 | Glazing Depth (mm) | Double | mm | Glazing depth |
| `frame_dimension_mm` | 12 | Frame Dimension (mm) | Double | mm | Frame dimension |
| `glass_weight_kg` | 13 | Glass Weight (kg) | Double | kg | Glass weight |
| `sealant_temp_c` | 14 | Sealant Temp (°C) | Double | °C | Sealant temperature |
| `press_force_n` | 15 | Press Force (N) | Double | N | Glass press force |
| `roller_speed_mm_s` | 16 | Roller Speed (mm/s) | Double | mm/s | Glazing roller speed |
| `gasket_compression_pct` | 17 | Gasket Compression (%) | Double | % | Gasket compression |
| `uv_cure_intensity_mw_cm2` | 18 | UV Cure Intensity (mW/cm²) | Double | mW/cm2 | UV cure intensity |
| `edge_clearance_mm` | 19 | Edge Clearance (mm) | Double | mm | Glass edge clearance |
| `moisture_level_pct` | 20 | Moisture Level (%) | Double | % | Moisture level |
| `alignment_offset_mm` | 21 | Alignment Offset (mm) | Double | mm | Alignment offset |

---

### Furniture

#### cnc_router

Simulates a CNC router for furniture manufacturing.

Source: `internal/machine/types/cnc_router.go`

| Name | NodeID | DisplayName | DataType | Unit | Description |
|------|--------|-------------|----------|------|-------------|
| `spindle_speed_rpm` | 7 | Spindle Speed (RPM) | Double | RPM | Spindle speed |
| `x_position_mm` | 8 | X Position (mm) | Double | mm | X axis position |
| `y_position_mm` | 9 | Y Position (mm) | Double | mm | Y axis position |
| `z_position_mm` | 10 | Z Position (mm) | Double | mm | Z axis position |
| `cut_depth_mm` | 11 | Cut Depth (mm) | Double | mm | Cut depth |
| `feed_rate_mm_min` | 12 | Feed Rate (mm/min) | Double | mm/min | Feed rate |
| `dust_extraction_m3_h` | 13 | Dust Extraction (m³/h) | Double | m3/h | Dust extraction airflow |
| `tool_wear_pct` | 14 | Tool Wear (%) | Double | % | Tool wear percentage |
| `power_consumption_kw` | 15 | Power Consumption (kW) | Double | kW | Power consumption |
| `vibration_mm_s` | 16 | Vibration (mm/s) | Double | mm/s | Vibration level |
| `vacuum_hold_down_kpa` | 17 | Vacuum Hold-Down (kPa) | Double | kPa | Vacuum hold-down pressure |
| `surface_finish_um` | 18 | Surface Finish (µm) | Double | um | Surface finish quality |
| `tool_temp_c` | 19 | Tool Temperature (°C) | Double | °C | Tool temperature |
| `coolant_flow_l_min` | 20 | Coolant Flow (L/min) | Double | L/min | Coolant flow rate |
| `material_removal_rate_cm3_min` | 21 | Material Removal Rate (cm³/min) | Double | cm3/min | Material removal rate |

#### edge_bander

Simulates a furniture edge banding machine.

Source: `internal/machine/types/edge_bander.go`

| Name | NodeID | DisplayName | DataType | Unit | Description |
|------|--------|-------------|----------|------|-------------|
| `glue_temp_c` | 7 | Glue Temperature (°C) | Double | °C | Glue pot temperature |
| `feed_speed_m_min` | 8 | Feed Speed (m/min) | Double | m/min | Feed speed |
| `trimmer_position_mm` | 9 | Trimmer Position (mm) | Double | mm | Trimmer position |
| `tape_tension_n` | 10 | Tape Tension (N) | Double | N | Edge tape tension |
| `roller_force_n` | 11 | Roller Force (N) | Double | N | Pressure roller force |
| `edge_thickness_mm` | 12 | Edge Thickness (mm) | Double | mm | Edge band thickness |
| `glue_flow_g_min` | 13 | Glue Flow (g/min) | Double | g/min | Glue flow rate |
| `panel_thickness_mm` | 14 | Panel Thickness (mm) | Double | mm | Panel thickness |
| `trim_quality_score_pct` | 15 | Trim Quality (%) | Double | % | Trim quality score |
| `pressure_zone_bar` | 16 | Pressure Zone (bar) | Double | bar | Pressure zone setting |
| `pre_milling_depth_mm` | 17 | Pre-Milling Depth (mm) | Double | mm | Pre-milling depth |
| `hot_air_temp_c` | 18 | Hot Air Temp (°C) | Double | °C | Hot air temperature |
| `joint_strength_n_cm` | 19 | Joint Strength (N/cm) | Double | N/cm | Joint strength |
| `scraper_position_mm` | 20 | Scraper Position (mm) | Double | mm | Scraper position |
| `buffing_speed_rpm` | 21 | Buffing Speed (RPM) | Double | RPM | Buffing unit speed |

#### assembly_press

Simulates a furniture assembly press (dowel/cam insertion).

Source: `internal/machine/types/assembly_press.go`

| Name | NodeID | DisplayName | DataType | Unit | Description |
|------|--------|-------------|----------|------|-------------|
| `press_force_kn` | 7 | Press Force (kN) | Double | kN | Press force |
| `alignment_accuracy_mm` | 8 | Alignment Accuracy (mm) | Double | mm | Alignment accuracy |
| `dwell_time_ms` | 9 | Dwell Time (ms) | Double | ms | Press dwell time |
| `cycle_pressure_bar` | 10 | Cycle Pressure (bar) | Double | bar | Cycle pressure |
| `ram_position_mm` | 11 | Ram Position (mm) | Double | mm | Ram position |
| `insertion_depth_mm` | 12 | Insertion Depth (mm) | Double | mm | Insertion depth |
| `parallelism_um` | 13 | Platen Parallelism (µm) | Double | um | Platen parallelism |
| `stroke_speed_mm_s` | 14 | Stroke Speed (mm/s) | Double | mm/s | Stroke speed |
| `part_presence_pct` | 15 | Part Presence (%) | Double | % | Part presence sensor |
| `force_curve_peak_kn` | 16 | Peak Force (kN) | Double | kN | Peak force in cycle |
| `work_energy_j` | 17 | Work Energy (J) | Double | J | Work energy per cycle |
| `temperature_c` | 18 | Temperature (°C) | Double | °C | Ambient temperature |
| `cushion_pressure_bar` | 19 | Cushion Pressure (bar) | Double | bar | Cushion pressure |
| `ejector_force_n` | 20 | Ejector Force (N) | Double | N | Ejector force |
| `servo_current_a` | 21 | Servo Current (A) | Double | A | Servo motor current |

---

### Metal Parts

#### laser_cutter

Simulates a fiber/CO2 laser cutting machine.

Source: `internal/machine/types/laser_cutter.go`

| Name | NodeID | DisplayName | DataType | Unit | Description |
|------|--------|-------------|----------|------|-------------|
| `laser_power_w` | 7 | Laser Power (W) | Double | W | Laser output power |
| `cutting_speed_mm_s` | 8 | Cutting Speed (mm/s) | Double | mm/s | Cutting speed |
| `assist_gas_pressure_bar` | 9 | Assist Gas Pressure (bar) | Double | bar | Assist gas pressure |
| `focal_point_mm` | 10 | Focal Point (mm) | Double | mm | Focal point position |
| `kerf_width_mm` | 11 | Kerf Width (mm) | Double | mm | Kerf width |
| `beam_quality_mm_mrad` | 12 | Beam Quality (mm·mrad) | Double | mm*mrad | Beam quality (BPP) |
| `nozzle_standoff_mm` | 13 | Nozzle Standoff (mm) | Double | mm | Nozzle standoff distance |
| `pierce_time_ms` | 14 | Pierce Time (ms) | Double | ms | Pierce time |
| `sheet_temp_c` | 15 | Sheet Temperature (°C) | Double | °C | Sheet temperature |
| `exhaust_flow_m3_h` | 16 | Exhaust Flow (m³/h) | Double | m3/h | Exhaust flow rate |
| `lens_condition_pct` | 17 | Lens Condition (%) | Double | % | Lens condition |
| `gas_consumption_l_min` | 18 | Gas Consumption (L/min) | Double | L/min | Gas consumption rate |
| `edge_roughness_um` | 19 | Edge Roughness (µm) | Double | um | Cut edge roughness |
| `taper_angle_deg` | 20 | Taper Angle (°) | Double | deg | Cut taper angle |
| `power_stability_pct` | 21 | Power Stability (%) | Double | % | Laser power stability |

#### press_brake

Simulates a hydraulic press brake for sheet metal bending.

Source: `internal/machine/types/press_brake.go`

| Name | NodeID | DisplayName | DataType | Unit | Description |
|------|--------|-------------|----------|------|-------------|
| `ram_force_tons` | 7 | Ram Force (tons) | Double | tons | Ram force |
| `bend_angle_deg` | 8 | Bend Angle (°) | Double | deg | Bend angle |
| `back_gauge_position_mm` | 9 | Back Gauge Position (mm) | Double | mm | Back gauge position |
| `stroke_length_mm` | 10 | Stroke Length (mm) | Double | mm | Stroke length |
| `die_temp_c` | 11 | Die Temperature (°C) | Double | °C | Die temperature |
| `crowning_mm` | 12 | Crowning (mm) | Double | mm | Crowning compensation |
| `daylight_mm` | 13 | Daylight Opening (mm) | Double | mm | Daylight opening |
| `ram_speed_mm_s` | 14 | Ram Speed (mm/s) | Double | mm/s | Ram speed |
| `tonnage_pct` | 15 | Tonnage Utilization (%) | Double | % | Tonnage utilization |
| `material_springback_deg` | 16 | Springback (°) | Double | deg | Material springback |
| `repeatability_mm` | 17 | Repeatability (mm) | Double | mm | Position repeatability |
| `hydraulic_pressure_bar` | 18 | Hydraulic Pressure (bar) | Double | bar | Hydraulic pressure |
| `beam_deflection_mm` | 19 | Beam Deflection (mm) | Double | mm | Beam deflection |
| `energy_consumption_kwh` | 20 | Energy Consumption (kWh) | Double | kWh | Energy consumption |
| `cycle_time_actual_s` | 21 | Actual Cycle Time (s) | Double | s | Actual cycle time |

#### deburring_machine

Simulates a brush or rotary deburring machine.

Source: `internal/machine/types/deburring_machine.go`

| Name | NodeID | DisplayName | DataType | Unit | Description |
|------|--------|-------------|----------|------|-------------|
| `brush_speed_rpm` | 7 | Brush Speed (RPM) | Double | RPM | Brush rotation speed |
| `contact_pressure_bar` | 8 | Contact Pressure (bar) | Double | bar | Contact pressure |
| `feed_rate_mm_s` | 9 | Feed Rate (mm/s) | Double | mm/s | Feed rate |
| `surface_finish_ra_um` | 10 | Surface Finish (µm Ra) | Double | um Ra | Surface finish |
| `material_removal_mm` | 11 | Material Removal (mm) | Double | mm | Material removal depth |
| `brush_wear_pct` | 12 | Brush Wear (%) | Double | % | Brush wear percentage |
| `power_consumption_kw` | 13 | Power Consumption (kW) | Double | kW | Power consumption |
| `vibration_mm_s` | 14 | Vibration (mm/s) | Double | mm/s | Vibration level |
| `coolant_flow_l_min` | 15 | Coolant Flow (L/min) | Double | L/min | Coolant flow rate |
| `dust_extraction_m3_h` | 16 | Dust Extraction (m³/h) | Double | m3/h | Dust extraction airflow |
| `part_temp_c` | 17 | Part Temperature (°C) | Double | °C | Part temperature |
| `spindle_current_a` | 18 | Spindle Current (A) | Double | A | Spindle motor current |
| `burr_height_before_um` | 19 | Burr Height Before (µm) | Double | um | Burr height before processing |
| `burr_height_after_um` | 20 | Burr Height After (µm) | Double | um | Burr height after processing |
| `cycle_count_brush` | 21 | Brush Cycle Count | Double | | Brush cycle count |

---

### Plastic Parts

#### injection_molding

Simulates a plastic injection molding machine.

Source: `internal/machine/types/injection_molding.go`

| Name | NodeID | DisplayName | DataType | Unit | Description |
|------|--------|-------------|----------|------|-------------|
| `barrel_zone1_temp_c` | 7 | Barrel Zone 1 Temp (°C) | Double | °C | Barrel zone 1 temperature (feed) |
| `barrel_zone2_temp_c` | 8 | Barrel Zone 2 Temp (°C) | Double | °C | Barrel zone 2 temperature |
| `barrel_zone3_temp_c` | 9 | Barrel Zone 3 Temp (°C) | Double | °C | Barrel zone 3 temperature |
| `barrel_zone4_temp_c` | 10 | Barrel Zone 4 Temp (°C) | Double | °C | Barrel zone 4 temperature (nozzle) |
| `injection_pressure_bar` | 11 | Injection Pressure (bar) | Double | bar | Injection pressure |
| `mold_temp_c` | 12 | Mold Temperature (°C) | Double | °C | Mold temperature |
| `clamp_force_tons` | 13 | Clamp Force (tons) | Double | tons | Clamp force |
| `cooling_time_s` | 14 | Cooling Time (s) | Double | s | Cooling time |
| `shot_weight_g` | 15 | Shot Weight (g) | Double | g | Shot weight |
| `cushion_mm` | 16 | Screw Cushion (mm) | Double | mm | Screw cushion |
| `screw_speed_rpm` | 17 | Screw Speed (RPM) | Double | RPM | Screw rotation speed |
| `back_pressure_bar` | 18 | Back Pressure (bar) | Double | bar | Back pressure |
| `melt_temp_c` | 19 | Melt Temperature (°C) | Double | °C | Melt temperature |
| `pack_pressure_bar` | 20 | Pack Pressure (bar) | Double | bar | Packing pressure |
| `fill_time_s` | 21 | Fill Time (s) | Double | s | Mold fill time |

#### extruder

Simulates a plastic extrusion machine.

Source: `internal/machine/types/extruder.go`

| Name | NodeID | DisplayName | DataType | Unit | Description |
|------|--------|-------------|----------|------|-------------|
| `zone1_temp_c` | 7 | Zone 1 Temp (°C) | Double | °C | Barrel zone 1 temperature (feed) |
| `zone2_temp_c` | 8 | Zone 2 Temp (°C) | Double | °C | Barrel zone 2 temperature |
| `zone3_temp_c` | 9 | Zone 3 Temp (°C) | Double | °C | Barrel zone 3 temperature |
| `zone4_temp_c` | 10 | Zone 4 Temp (°C) | Double | °C | Barrel zone 4 temperature (die) |
| `screw_speed_rpm` | 11 | Screw Speed (RPM) | Double | RPM | Screw rotation speed |
| `melt_pressure_bar` | 12 | Melt Pressure (bar) | Double | bar | Melt pressure |
| `die_pressure_bar` | 13 | Die Pressure (bar) | Double | bar | Die pressure |
| `haul_off_speed_m_min` | 14 | Haul-Off Speed (m/min) | Double | m/min | Haul-off speed |
| `melt_temp_c` | 15 | Melt Temperature (°C) | Double | °C | Melt temperature |
| `motor_current_a` | 16 | Motor Current (A) | Double | A | Motor current draw |
| `throughput_kg_h` | 17 | Throughput (kg/h) | Double | kg/h | Material throughput |
| `torque_pct` | 18 | Torque Utilization (%) | Double | % | Screw torque utilization |
| `head_pressure_bar` | 19 | Head Pressure (bar) | Double | bar | Head pressure |
| `vacuum_mbar` | 20 | Vacuum Level (mbar) | Double | mbar | Vacuum level |
| `cooling_water_temp_c` | 21 | Cooling Water Temp (°C) | Double | °C | Cooling water temperature |
| `line_speed_m_min` | 22 | Line Speed (m/min) | Double | m/min | Line speed |

#### trimming_press

Simulates a plastic part trimming press.

Source: `internal/machine/types/trimming_press.go`

| Name | NodeID | DisplayName | DataType | Unit | Description |
|------|--------|-------------|----------|------|-------------|
| `press_force_kn` | 7 | Press Force (kN) | Double | kN | Press force |
| `trim_accuracy_mm` | 8 | Trim Accuracy (mm) | Double | mm | Trim accuracy |
| `feed_rate_mm_s` | 9 | Feed Rate (mm/s) | Double | mm/s | Feed rate |
| `die_clearance_mm` | 10 | Die Clearance (mm) | Double | mm | Die clearance |
| `stroke_speed_spm` | 11 | Stroke Speed (SPM) | Double | SPM | Stroke speed |
| `material_thickness_mm` | 12 | Material Thickness (mm) | Double | mm | Material thickness |
| `die_temp_c` | 13 | Die Temperature (°C) | Double | °C | Die temperature |
| `stripper_force_n` | 14 | Stripper Force (N) | Double | N | Stripper force |
| `slug_ejection_pct` | 15 | Slug Ejection (%) | Double | % | Slug ejection rate |
| `lubrication_level_pct` | 16 | Lubrication Level (%) | Double | % | Lubrication level |
| `die_wear_pct` | 17 | Die Wear (%) | Double | % | Die wear percentage |
| `punch_alignment_um` | 18 | Punch Alignment (µm) | Double | um | Punch alignment |
| `press_energy_j` | 19 | Press Energy (J) | Double | J | Press energy per stroke |
| `vibration_mm_s` | 20 | Vibration (mm/s) | Double | mm/s | Vibration level |
| `noise_level_db` | 21 | Noise Level (dB) | Double | dB | Noise level |
