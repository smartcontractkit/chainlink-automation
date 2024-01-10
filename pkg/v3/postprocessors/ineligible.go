package postprocessors

import (
	"context"
	"errors"

	"github.com/smartcontractkit/chainlink-automation/pkg/v3/telemetry"
	ocr2keepers "github.com/smartcontractkit/chainlink-automation/pkg/v3/types"
)

type ineligiblePostProcessor struct {
	lggr         *telemetry.Logger
	stateUpdater ocr2keepers.UpkeepStateUpdater
}

func NewIneligiblePostProcessor(stateUpdater ocr2keepers.UpkeepStateUpdater, logger *telemetry.Logger) *ineligiblePostProcessor {
	return &ineligiblePostProcessor{
		lggr:         telemetry.WrapTelemetryLogger(logger, "ineligible-post-processor"),
		stateUpdater: stateUpdater,
	}
}

func (p *ineligiblePostProcessor) PostProcess(ctx context.Context, results []ocr2keepers.CheckResult, _ []ocr2keepers.UpkeepPayload) error {
	var merr error
	ineligible := 0
	for _, res := range results {
		if res.PipelineExecutionState == 0 && !res.Eligible {
			if err := p.lggr.Collect(res.WorkID, uint64(res.Trigger.BlockNumber), telemetry.Completed); err != nil {
				p.lggr.Println(err.Error())
			}

			err := p.stateUpdater.SetUpkeepState(ctx, res, ocr2keepers.Ineligible)
			if err != nil {
				merr = errors.Join(merr, err)
				continue
			}
			ineligible++
		}
	}
	p.lggr.Printf("post-processing %d results, %d ineligible\n", len(results), ineligible)
	return merr
}
