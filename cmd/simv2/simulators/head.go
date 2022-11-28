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
	go func() {
		for range ticker.C {
			cb(ktypes.BlockKey(big.NewInt(initialBlock).String()))
			initialBlock++
		}
	}()

	return nil
}
