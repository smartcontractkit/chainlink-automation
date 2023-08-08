package simulators

import (
	"context"
	"fmt"
	"math/big"
	"sync"

	"go.uber.org/multierr"

	"github.com/smartcontractkit/ocr2keepers/pkg/v3/plugin"
	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg/v3/types"
)

type SimulatedUpkeep struct {
	ID         *big.Int
	EligibleAt []*big.Int
	Performs   map[string]ocr2keepers.TransmitEvent // performs at block number
}

func (ct *SimulatedContract) GetActiveUpkeepIDs(ctx context.Context) ([]ocr2keepers.UpkeepIdentifier, error) {

	ct.mu.RLock()
	ct.logger.Printf("getting keys at block %s", ct.lastBlock)

	keys := []ocr2keepers.UpkeepIdentifier{}

	// TODO: filter out cancelled upkeeps
	for key := range ct.upkeeps {
		b := [32]byte{}
		copy(b[:], []byte(key))
		keys = append(keys, ocr2keepers.UpkeepIdentifier(b))
	}
	ct.mu.RUnlock()

	// call to GetState
	err := <-ct.rpc.Call(ctx, "getState")
	if err != nil {
		return nil, err
	}
	// call to GetActiveIDs
	// TODO: batch size is hard coded at 10_000, if the number of keys is more
	// than this, simulate another rpc call
	err = <-ct.rpc.Call(ctx, "getActiveIDs")
	if err != nil {
		return nil, err
	}

	return keys, nil
}

func (ct *SimulatedContract) CheckUpkeeps(ctx context.Context, payloads ...ocr2keepers.UpkeepPayload) ([]ocr2keepers.CheckResult, error) {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	var (
		mErr    error
		wg      sync.WaitGroup
		results = make([]ocr2keepers.CheckResult, len(payloads))
	)

	for i, payload := range payloads {
		wg.Add(1)
		go func(i int, key ocr2keepers.UpkeepPayload, en plugin.Encoder) {
			defer wg.Done()

			block := new(big.Int).SetInt64(int64(key.Trigger.BlockNumber))

			up, ok := ct.upkeeps[key.UpkeepID.String()]
			if !ok {
				mErr = multierr.Append(mErr, fmt.Errorf("upkeep not registered"))
				return
			}

			results[i] = ocr2keepers.CheckResult{
				Eligible:     false,
				Retryable:    false,
				GasAllocated: 5_000_000, // TODO: make this configurable
				UpkeepID:     key.UpkeepID,
				Trigger:      key.Trigger,
				PerformData:  []byte{}, // TODO: add perform data from configuration
			}

			// start at the highest blocks eligible. the first eligible will be a block
			// lower than the current
			for j := len(up.EligibleAt) - 1; j >= 0; j-- {
				e := up.EligibleAt[j]

				if block.Cmp(e) >= 0 {
					results[i].Eligible = true

					// check that upkeep has not been recently performed between two
					// points of eligibility
					// is there a log between eligible and block
					var t int64
					diff := new(big.Int).Sub(block, e).Int64()
					for t = 0; t <= diff; t++ {
						c := new(big.Int).Add(e, big.NewInt(t))
						_, ok := up.Performs[c.String()]
						if ok {
							results[i].Eligible = false
							return
						}
					}

					return
				}
			}
		}(i, payload, ct.enc)
	}

	wg.Wait()

	if mErr != nil {
		return nil, mErr
	}

	// call to CheckUpkeep
	err := <-ct.rpc.Call(ctx, "checkUpkeep")
	if err != nil {
		return nil, err
	}

	// call to SimulatePerform
	err = <-ct.rpc.Call(ctx, "simulatePerform")
	if err != nil {
		return nil, err
	}

	output := make([]ocr2keepers.CheckResult, len(results))
	copy(output, results)

	return output, nil
}
