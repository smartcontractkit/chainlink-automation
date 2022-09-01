package keepers

import (
	"context"

	"github.com/smartcontractkit/ocr2keepers/ocr2keepers/types"
)

type SampleRatio float32

func (r SampleRatio) OfInt(count int) int {
	// calculates a simple floor operation
	return int(float32(r) * float32(count))
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

type UpkeepService interface {
	SampleUpkeeps(context.Context) ([]types.UpkeepResult, error)
	CheckUpkeep(context.Context, types.UpkeepKey) (types.UpkeepResult, error)
	SetUpkeepState(context.Context, types.UpkeepKey, types.UpkeepState) error
}
