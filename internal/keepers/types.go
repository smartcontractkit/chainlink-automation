package keepers

import (
	"context"
	"math"

	"github.com/smartcontractkit/ocr2keepers/pkg/types"
)

type SampleRatio float32

func (r SampleRatio) OfInt(count int) int {
	// rounds the result using basic rounding op
	return int(math.Round(float64(r) * float64(count)))
}

type Upkeep struct {
	PerformData []byte
}

const (
	Eligible types.UpkeepState = iota
	Skip
	Perform
	Reported
)

type upkeepService interface {
	SampleUpkeeps(context.Context) ([]*types.UpkeepResult, error)
	CheckUpkeep(context.Context, types.UpkeepKey) (types.UpkeepResult, error)
	SetUpkeepState(context.Context, types.UpkeepKey, types.UpkeepState) error
}
