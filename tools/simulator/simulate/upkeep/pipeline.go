package upkeep

import (
	"context"
	"log"
	"math/big"
	"sync"

	ocr2keepers "github.com/smartcontractkit/chainlink-automation/pkg/v3/types"
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

func (p *CheckPipeline) CheckUpkeeps(ctx context.Context, payloads ...ocr2keepers.UpkeepPayload) ([]ocr2keepers.CheckResult, error) {
	var (
		mErr    error
		wg      sync.WaitGroup
		results = make([]ocr2keepers.CheckResult, len(payloads))
	)

	for idx, payload := range payloads {
		wg.Add(1)
		go func(resultIdx int, key ocr2keepers.UpkeepPayload) {
			defer wg.Done()

			p.checkTelemetry.CheckID(key.UpkeepID.String(), uint64(key.Trigger.BlockNumber), key.Trigger.BlockHash)

			results[resultIdx] = p.makeResult(key)
		}(idx, payload)
	}

	wg.Wait()

	if mErr != nil {
		return nil, mErr
	}

	// call to CheckUpkeep
	err := <-p.rpc.Call(ctx, "checkUpkeep")
	if err != nil {
		return nil, err
	}

	// call to SimulatePerform
	err = <-p.rpc.Call(ctx, "simulatePerform")
	if err != nil {
		return nil, err
	}

	output := make([]ocr2keepers.CheckResult, len(results))
	copy(output, results)

	return output, nil
}

func (p *CheckPipeline) makeResult(payload ocr2keepers.UpkeepPayload) ocr2keepers.CheckResult {
	result := ocr2keepers.CheckResult{
		PipelineExecutionState: 0,
		Retryable:              false,
		Eligible:               false,
		IneligibilityReason:    0,
		UpkeepID:               payload.UpkeepID,
		Trigger:                payload.Trigger,
		WorkID:                 payload.WorkID,
		GasAllocated:           5_000_000, // TODO: make this configurable
		PerformData:            []byte{},  // TODO: add perform data from configuration
		FastGasWei:             big.NewInt(1_000_000),
		LinkNative:             big.NewInt(1_000_000),
	}

	block := new(big.Int).SetInt64(int64(payload.Trigger.BlockNumber))
	performs := p.performs.PerformsForUpkeepID(payload.UpkeepID.String())

	if simulated, ok := p.active.GetByUpkeepID(payload.UpkeepID); ok {
		if simulated.AlwaysEligible {
			result.Eligible = true
		} else {
			result.Eligible = isEligible(simulated.EligibleAt, performs, block)
		}

		execState := uint8(simulated.States.GetNextState())
		if execState != 0 {
			result.PipelineExecutionState = execState
			result.Retryable = simulated.Retryable
		}

		p.logger.Printf("%s eligibility %t at block %d", payload.UpkeepID, result.Eligible, block)
	} else {
		p.logger.Printf("%s is not active", payload.UpkeepID)
	}

	return result
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
