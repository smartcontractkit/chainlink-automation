package upkeep

import (
	"context"
	"log"
	"math/big"

	"github.com/smartcontractkit/ocr2keepers/cmd/simv3/simulator/chain"
	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg/v3/types"
)

type CheckPipeline struct {
	// provided dependencies
	listener *chain.Listener
	active   *ActiveTracker
	logger   *log.Logger
}

// TODO: provide upkeep configurations to this component
// NewCheckPipeline ...
func NewCheckPipeline(listener *chain.Listener, active *ActiveTracker, logger *log.Logger) *CheckPipeline {
	return &CheckPipeline{
		listener: listener,
		active:   active,
		logger:   logger,
	}
}

// TODO: finish retry/error conditions in check pipeline
// CheckUpkeeps simulates a check pipeline run and may return whether a result
// is eligible or retryable based on pipeline return cases. Multiple assumptions
// are made within this simulation and any changes to the production pipeline
// should be reflected here.
func (cp *CheckPipeline) CheckUpkeeps(ctx context.Context, payloads ...ocr2keepers.UpkeepPayload) ([]ocr2keepers.CheckResult, error) {
	results := make([]ocr2keepers.CheckResult, len(payloads))

	for i, payload := range payloads {

		// 1. verify check block and hash are valid
		// 1a. check block too old: no failure reason, not retryable, not eligible,

		// 1. upkeep active status
		// _, ok := cp.active.GetByID(payload.UpkeepID.BigInt())
		//if !ok {
		results[i] = ocr2keepers.CheckResult{
			PipelineExecutionState: 0,
			Retryable:              false,
			Eligible:               true,
			IneligibilityReason:    0,
			UpkeepID:               payload.UpkeepID,
			Trigger:                payload.Trigger,
			WorkID:                 payload.WorkID,
			GasAllocated:           5_000_000, // TODO: update from config
			PerformData:            []byte{},  // TODO: update from config
			FastGasWei:             big.NewInt(1_000_000),
			LinkNative:             big.NewInt(1_000_000),
		}
		//}

		// 2. log triggered status; was the payload triggered by a log (if log trigger type)
		// 3. eligibility status
		// 4. performed status
	}

	return results, nil
}

/*


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
		go func(i int, key ocr2keepers.UpkeepPayload, en ocr2keepers.Encoder) {
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
*/
