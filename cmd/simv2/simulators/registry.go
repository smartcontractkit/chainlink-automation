package simulators

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"sync"

	"go.uber.org/multierr"

	"github.com/smartcontractkit/ocr2keepers/pkg/types"
)

type SimulatedUpkeep struct {
	ID         *big.Int
	EligibleAt []*big.Int
	Performs   map[string]types.PerformLog // performs at block number
}

func (ct *SimulatedContract) GetActiveUpkeepKeys(ctx context.Context, key types.BlockKey) ([]types.UpkeepKey, error) {

	ct.mu.RLock()
	ct.logger.Printf("getting keys at block %s", ct.lastBlock)

	block := ct.lastBlock.String()
	keys := []types.UpkeepKey{}

	// TODO: filter out cancelled upkeeps
	for key := range ct.upkeeps {
		k := types.UpkeepKey([]byte(fmt.Sprintf("%s|%s", block, key)))
		keys = append(keys, k)
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

func (ct *SimulatedContract) CheckUpkeep(ctx context.Context, keys ...types.UpkeepKey) (types.UpkeepResults, error) {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	var (
		mErr    error
		wg      sync.WaitGroup
		results = make(types.UpkeepResults, len(keys))
	)

	for i, key := range keys {
		wg.Add(1)
		go func(i int, key types.UpkeepKey) {
			defer wg.Done()

			parts := strings.Split(string(key), "|")
			if len(parts) != 2 {
				panic("upkeep key does not contain block and id")
			}

			block, ok := new(big.Int).SetString(parts[0], 10)
			if !ok {
				mErr = multierr.Append(mErr, fmt.Errorf("block in key not parsable as big int"))
				return
			}

			up, ok := ct.upkeeps[parts[1]]
			if !ok {
				mErr = multierr.Append(mErr, fmt.Errorf("upkeep not registered"))
				return
			}

			var bl [32]byte
			results[i] = types.UpkeepResult{
				Key:     key,
				State:   types.NotEligible,
				GasUsed: big.NewInt(0),
				/*
					FailureReason    uint8
				*/
				PerformData:      []byte{}, // TODO: add perform data
				FastGasWei:       big.NewInt(0),
				LinkNative:       big.NewInt(0),
				CheckBlockNumber: uint32(block.Int64() - 1), // minus 1 because the real contract does this
				CheckBlockHash:   bl,
			}

			// call to CheckUpkeep
			err := <-ct.rpc.Call(ctx, "checkUpkeep")
			if err != nil {
				mErr = multierr.Append(mErr, err)
				return
			}

			// call to SimulatePerform
			err = <-ct.rpc.Call(ctx, "simulatePerform")
			if err != nil {
				mErr = multierr.Append(mErr, err)
				return
			}

			// call telemetry after RPC delays have been applied. if a check is cancelled
			// it doesn't count toward telemetry.
			ct.telemetry.CheckKey([]byte(key))

			// start at the highest blocks eligible. the first eligible will be a block
			// lower than the current
			for j := len(up.EligibleAt) - 1; j >= 0; j-- {
				e := up.EligibleAt[j]

				if block.Cmp(e) >= 0 {
					results[i].State = types.Eligible

					// check that upkeep has not been recently performed between two
					// points of eligibility
					// is there a log between eligible and block
					var t int64
					diff := new(big.Int).Sub(block, e).Int64()
					for t = 0; t <= diff; t++ {
						c := new(big.Int).Add(e, big.NewInt(t))
						_, ok := up.Performs[c.String()]
						if ok {
							results[i].State = types.NotEligible
							return
						}
					}

					return
				}
			}

			return
		}(i, key)
	}

	wg.Wait()

	return results, nil
}

func (ct *SimulatedContract) IdentifierFromKey(key types.UpkeepKey) (types.UpkeepIdentifier, error) {
	parts := strings.Split(string(key), "|")
	if len(parts) != 2 {
		panic("upkeep key does not contain block and id")
	}

	return types.UpkeepIdentifier(parts[1]), nil
}
