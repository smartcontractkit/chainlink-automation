package simulators

import (
	"context"

	"github.com/smartcontractkit/ocr2keepers/pkg/chain"
	ktypes "github.com/smartcontractkit/ocr2keepers/pkg/types"
)

func (ct *SimulatedContract) HeadTicker() chan ktypes.BlockKey {
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
			send(ct.chHeads, chain.BlockKey(block.BlockNumber.String()))
		}
	}
}

// send does a non-blocking send of the key on c.
func send(c chan ktypes.BlockKey, k ktypes.BlockKey) {
	select {
	case c <- k:
	default:
	}
}
