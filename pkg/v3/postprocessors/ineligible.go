package postprocessors

import (
	"context"
	"errors"
	"log"

	"github.com/smartcontractkit/ocr2keepers/pkg/v3/telemetry"
	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg/v3/types"
)

type ineligiblePostProcessor struct {
	lggr         *log.Logger
	stateUpdater ocr2keepers.UpkeepStateUpdater
}

func NewInEligiblePostProcessor(stateUpdater ocr2keepers.UpkeepStateUpdater, logger *log.Logger) *ineligiblePostProcessor {
	return &ineligiblePostProcessor{
		lggr:         telemetry.WrapLogger(logger, "InEligiblePostProcessor"),
		stateUpdater: stateUpdater,
	}
}

func (p *ineligiblePostProcessor) PostProcess(ctx context.Context, results []ocr2keepers.CheckResult, _ []ocr2keepers.UpkeepPayload) error {
	var merr error
	ineligible := 0
	for _, res := range results {
		if !res.Eligible {
			ineligible++
			merr = errors.Join(merr, p.stateUpdater.SetUpkeepState(ctx, res, ocr2keepers.Ineligible))
		}
	}
	p.lggr.Printf("post-processing %d results, %d ineligible\n", len(results), ineligible)
	return merr
}
