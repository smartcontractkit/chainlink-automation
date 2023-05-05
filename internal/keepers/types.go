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
	SampleUpkeeps(context.Context, ...func(types.UpkeepKey) bool) (types.BlockKey, types.UpkeepResults, error)
	CheckUpkeep(context.Context, bool, ...types.UpkeepKey) (types.UpkeepResults, error)
}
type Coordinator interface {
	IsPending(key types.UpkeepKey) bool
	Accept(keys types.UpkeepKey) error
	IsTransmissionConfirmed(key types.UpkeepKey) bool
}
