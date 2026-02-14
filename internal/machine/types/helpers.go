package types

// clampF clamps a float64 pointer value between min and max.
func clampF(v *float64, min, max float64) {
	if *v < min {
		*v = min
	} else if *v > max {
		*v = max
	}
}
