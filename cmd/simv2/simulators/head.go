package simulators

import (
	"context"
	"math/big"
	"time"

	ktypes "github.com/smartcontractkit/ocr2keepers/pkg/types"
)

func (ct *SimulatedContract) OnNewHead(ctx context.Context, cb func(header ktypes.BlockKey)) error {
	initialBlock := int64(0)
	ticker := time.NewTicker(time.Second * 2)
	for range ticker.C {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			cb(ktypes.BlockKey(big.NewInt(initialBlock).String()))
			initialBlock++
		}
	}
	return nil
}
