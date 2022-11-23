package simulators

import (
	"context"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/core/types"
)

type simulatedSubscription struct {
}

func (s *simulatedSubscription) Unsubscribe() {

}

func (s *simulatedSubscription) Err() <-chan error {
	return make(chan error)
}

func (ct *SimulatedContract) SubscribeNewHead(ctx context.Context, ch chan<- *types.Header) (ethereum.Subscription, error) {
	initialBlock := int64(0)
	ticker := time.NewTicker(time.Second * 2)
	go func() {
		for range ticker.C {
			ch <- &types.Header{
				Number: big.NewInt(initialBlock),
			}
			initialBlock++
		}
	}()

	return &simulatedSubscription{}, nil
}
