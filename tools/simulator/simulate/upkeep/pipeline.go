package upkeep

import (
	"context"
	"log"
	"math/big"
	"sync"

	ocr2keepers "github.com/smartcontractkit/chainlink-common/pkg/types/automation"
	"github.com/smartcontractkit/chainlink-automation/tools/simulator/config"
	"github.com/smartcontractkit/chainlink-automation/tools/simulator/simulate/net"
)

type CheckTelemetry interface {
	CheckID(string, uint64, [32]byte)
}

type CheckPipeline struct {
	// provided dependencies
	active         *ActiveTracker
	performs       *PerformTracker
	rpc            *net.SimulatedNetworkService
	checkTelemetry CheckTelemetry
	logger         *log.Logger
}

// TODO: provide upkeep configurations to this component
// NewCheckPipeline ...
func NewCheckPipeline(
	conf config.SimulationPlan,
	active *ActiveTracker,
	performs *PerformTracker,
	netTelemetry net.NetTelemetry,
	conTelemetry CheckTelemetry,
	logger *log.Logger,
) *CheckPipeline {
	service := net.NewSimulatedNetworkService(
		conf.RPC.ErrorRate,
		conf.RPC.RateLimitThreshold,
		conf.RPC.AverageLatency,
		netTelemetry,
	)

	return &CheckPipeline{
		active:         active,
		performs:       performs,
		rpc:            service,
		checkTelemetry: conTelemetry,
		logger:         log.New(logger.Writer(), "[check-pipeline] ", log.Ldate|log.Ltime|log.Lshortfile),
	}
}

// TODO: finish retry/error conditions in check pipeline
// CheckUpkeeps simulates a check pipeline run and may return whether a result
// is eligible or retryable based on pipeline return cases. Multiple assumptions
// are made within this simulation and any changes to the production pipeline
// should be reflected here.
/*
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
*/

func (cp *CheckPipeline) CheckUpkeeps(ctx context.Context, payloads ...ocr2keepers.UpkeepPayload) ([]ocr2keepers.CheckResult, error) {
	var (
		mErr    error
		wg      sync.WaitGroup
		results = make([]ocr2keepers.CheckResult, len(payloads))
	)

	for idx, payload := range payloads {
		wg.Add(1)
		go func(resultIdx int, key ocr2keepers.UpkeepPayload) {
			defer wg.Done()

			block := new(big.Int).SetInt64(int64(key.Trigger.BlockNumber))
			performs := cp.performs.PerformsForUpkeepID(key.UpkeepID.String())

			cp.checkTelemetry.CheckID(key.UpkeepID.String(), uint64(key.Trigger.BlockNumber), key.Trigger.BlockHash)

			results[resultIdx] = ocr2keepers.CheckResult{
				PipelineExecutionState: 0,
				Retryable:              false,
				Eligible:               false,
				IneligibilityReason:    0,
				UpkeepID:               key.UpkeepID,
				Trigger:                key.Trigger,
				WorkID:                 key.WorkID,
				GasAllocated:           5_000_000, // TODO: make this configurable
				PerformData:            []byte{},  // TODO: add perform data from configuration
				FastGasWei:             big.NewInt(1_000_000),
				LinkNative:             big.NewInt(1_000_000),
			}

			if simulated, ok := cp.active.GetByUpkeepID(key.UpkeepID); ok {
				if simulated.AlwaysEligible {
					results[resultIdx].Eligible = true
				} else {
					results[resultIdx].Eligible = isEligible(simulated.EligibleAt, performs, block)
				}

				cp.logger.Printf("%s eligibility %t at block %d", key.UpkeepID, results[resultIdx].Eligible, block)
			} else {
				cp.logger.Printf("%s is not active", key.UpkeepID)
			}
		}(idx, payload)
	}

	wg.Wait()

	if mErr != nil {
		return nil, mErr
	}

	// call to CheckUpkeep
	err := <-cp.rpc.Call(ctx, "checkUpkeep")
	if err != nil {
		return nil, err
	}

	// call to SimulatePerform
	err = <-cp.rpc.Call(ctx, "simulatePerform")
	if err != nil {
		return nil, err
	}

	output := make([]ocr2keepers.CheckResult, len(results))
	copy(output, results)

	return output, nil
}

func isEligible(eligibles, performs []*big.Int, block *big.Int) bool {
	var eligible bool

	// start at the highest blocks eligible. the first eligible will be a block
	// lower than the current
	for eligibleIdx := len(eligibles) - 1; eligibleIdx >= 0; eligibleIdx-- {
		eligibleBlock := eligibles[eligibleIdx]

		if block.Cmp(eligibleBlock) >= 0 {
			// check that upkeep has not been recently performed between two
			// points of eligibility
			// is there a log between eligible and block
			blockRange := new(big.Int).Sub(block, eligibleBlock).Int64()

			for rangePoint := int64(0); rangePoint <= blockRange; rangePoint++ {
				checkBlock := new(big.Int).Add(eligibleBlock, big.NewInt(rangePoint))

				for _, performBlock := range performs {
					if performBlock.Cmp(checkBlock) == 0 {
						return false
					}
				}
			}

			return true
		}
	}

	return eligible
}
