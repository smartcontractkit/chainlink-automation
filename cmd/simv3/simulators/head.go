package simulators

import (
	"context"

	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg/v3/types"
)

func (ct *SimulatedContract) HeadTicker() chan ocr2keepers.BlockKey {
	return ct.chHeads
}

func (ct *SimulatedContract) forwardHeads(ctx context.Context) {
	sub, blocksCh := ct.src.Subscribe(false)
	defer ct.src.Unsubscribe(sub)

	for {
		select {
		case <-ct.done:
			return
		case <-ctx.Done():
			return
		case block := <-blocksCh:
			send(ct.chHeads, ocr2keepers.BlockKey(block.BlockNumber.String()))
		}
	}
}

// send does a non-blocking send of the key on c.
func send(c chan ocr2keepers.BlockKey, k ocr2keepers.BlockKey) {
	select {
	case c <- k:
	default:
	}
}
