package observer

import (
	"context"
	"log"

	"github.com/smartcontractkit/ocr2keepers/pkg/types"
)

type Observer interface {
	Observe() (types.BlockKey, []types.UpkeepIdentifier, error)
	CheckUpkeep(ctx context.Context, keys ...types.UpkeepKey) ([]types.UpkeepResult, error)
	Start()
	Stop()
	SetLogger(logger *log.Logger)
}
