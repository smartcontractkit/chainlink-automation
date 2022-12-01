package simulators

import (
	"context"

	ktypes "github.com/smartcontractkit/ocr2keepers/pkg/types"
)

func (ct *SimulatedContract) OnNewHead(ctx context.Context, cb func(header ktypes.BlockKey)) error {
	sub, blocksCh := ct.src.Subscribe(false)
	defer ct.src.Unsubscribe(sub)

	for {
		select {
		case <-ct.done:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		case block := <-blocksCh:
			cb(ktypes.BlockKey(block.BlockNumber.String()))
		}
	}
}
