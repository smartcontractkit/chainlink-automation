package postprocessors

import (
	"context"
	"errors"
	"log"

	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg/v3/types"
)

type ineligiblePostProcessor struct {
	lggr         *log.Logger
	stateUpdater ocr2keepers.UpkeepStateUpdater
}

func NewIneligiblePostProcessor(stateUpdater ocr2keepers.UpkeepStateUpdater, logger *log.Logger) *ineligiblePostProcessor {
	return &ineligiblePostProcessor{
		lggr:         logger,
		stateUpdater: stateUpdater,
	}
}

func (p *ineligiblePostProcessor) PostProcess(ctx context.Context, results []ocr2keepers.CheckResult, _ []ocr2keepers.UpkeepPayload) error {
	var merr error
	ineligible := 0
	for _, res := range results {
		if res.PipelineExecutionState == 0 && !res.Eligible {
			err := p.stateUpdater.SetUpkeepState(ctx, res, ocr2keepers.Ineligible)
			if err != nil {
				ineligible++
			}
			merr = errors.Join(merr, err)
		}
	}
	p.lggr.Printf("[ineligible-post-processor] post-processing %d results, %d ineligible\n", len(results), ineligible)
	return merr
}
