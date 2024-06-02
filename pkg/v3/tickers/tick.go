package tickers

import (
	"context"
	common "github.com/smartcontractkit/chainlink-common/pkg/types/automation"
)

// Tick is the container for the individual tick
type Tick interface {
	// Value provides data scoped to the tick
	Value(ctx context.Context) ([]common.UpkeepPayload, error)
}
