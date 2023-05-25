package keepers

import (
	"context"

	"github.com/smartcontractkit/ocr2keepers/pkg/types"
)

type upkeepService interface {
	SampleUpkeeps(context.Context, ...func(types.UpkeepKey) bool) (types.BlockKey, types.UpkeepResults, error)
	CheckUpkeep(context.Context, bool, ...types.UpkeepKey) (types.UpkeepResults, error)
}
type Coordinator interface {
	IsPending(key types.UpkeepKey) bool
	Accept(keys types.UpkeepKey) error
	IsTransmissionConfirmed(key types.UpkeepKey) bool
}

type Observer interface {
	Observe() (types.BlockKey, []types.UpkeepIdentifier, error)
	CheckUpkeep(ctx context.Context, keys ...types.UpkeepKey) ([]types.UpkeepResult, error)
	Start()
	Stop()
}
