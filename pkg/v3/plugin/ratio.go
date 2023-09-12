package plugin

import (
	"fmt"
	"math"
)

type sampleRatio float32

func (r sampleRatio) OfInt(count int) int {
	if count == 0 {
		return 0
	}

	// rounds the result using basic rounding op
	value := math.Round(float64(r) * float64(count))
	if value < 1.0 {
		return 1
	}

	return int(value)
}

func (r sampleRatio) String() string {
	return fmt.Sprintf("%.8f", float32(r))
}
