package observer

import (
	"context"
	"time"

	"github.com/smartcontractkit/ocr2keepers/pkg/ratio"
	"github.com/smartcontractkit/ocr2keepers/pkg/types"
)

type Observer interface {
	Observe() (types.BlockKey, []types.UpkeepIdentifier, error)
	CheckUpkeep(ctx context.Context, keys ...types.UpkeepKey) ([]types.UpkeepResult, error)
	Start()
	Stop()
	SetSamplingRatio(ratio ratio.SampleRatio)
	SetSamplingDuration(duration time.Duration)
	SetMercuryLookup(mercuryLookup bool)
}
