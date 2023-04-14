package ratio

import (
	"fmt"
	"math"
)

type SampleRatio float32

func (r SampleRatio) OfInt(count int) int {
	// rounds the result using basic rounding op
	return int(math.Round(float64(r) * float64(count)))
}

func (r SampleRatio) String() string {
	return fmt.Sprintf("%.8f", float32(r))
}
