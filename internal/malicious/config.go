package malicious

type ConfigRemapping struct {
	MaxObservationLength int
}

// NoOpConfigModifier returns the config remapping received without altering in
func NoOpConfigModifier(m ConfigRemapping) (string, ConfigRemapping) {
	return "No-op config manipulator", m
}

// IncreaseMaxObservationLength doubles the max observation length
func IncreaseMaxObservationLength(m ConfigRemapping) (string, ConfigRemapping) {
	return "Increase MaxObservationLength", ConfigRemapping{MaxObservationLength: m.MaxObservationLength * 2}
}
