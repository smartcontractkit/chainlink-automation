package keepers

import (
	"context"
	"fmt"
	"math"

	"github.com/smartcontractkit/ocr2keepers/pkg/types"
)

type sampleRatio float32

func (r sampleRatio) OfInt(count int) int {
	// rounds the result using basic rounding op
	return int(math.Round(float64(r) * float64(count)))
}

func (r sampleRatio) String() string {
	return fmt.Sprintf("%.8f", float32(r))
}

type upkeepService interface {
	SampleUpkeeps(context.Context, ...func(types.UpkeepKey) bool) ([]*types.UpkeepResult, error)
	CheckUpkeep(context.Context, types.UpkeepKey) (types.UpkeepResult, error)
}

type filterer interface {
	Filter() func(types.UpkeepKey) bool
	Accept(key types.UpkeepKey) error
	IsTransmissionConfirmed(key types.UpkeepKey) bool
}
