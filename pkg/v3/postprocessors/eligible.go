package postprocessors

import (
	"context"

	"github.com/smartcontractkit/chainlink-automation/pkg/v3/telemetry"
	ocr2keepers "github.com/smartcontractkit/chainlink-automation/pkg/v3/types"
)

// checkResultAdder is a general interface for a result store that accepts check results
type checkResultAdder interface {
	// Add inserts the provided check result in the store
	Add(...ocr2keepers.CheckResult)
}

// PostProcessor is the general interface for a processing function after checking eligibility
// status
type PostProcessor interface {
	// PostProcess takes a slice of results where eligibility status is known
	PostProcess(context.Context, []ocr2keepers.CheckResult, []ocr2keepers.UpkeepPayload) error
}

type eligiblePostProcessor struct {
	lggr         *telemetry.Logger
	resultsAdder checkResultAdder
}

func NewEligiblePostProcessor(resultsAdder checkResultAdder, logger *telemetry.Logger) *eligiblePostProcessor {
	return &eligiblePostProcessor{
		lggr:         telemetry.WrapTelemetryLogger(logger, "eligible-post-processor"),
		resultsAdder: resultsAdder,
	}
}

func (p *eligiblePostProcessor) PostProcess(_ context.Context, results []ocr2keepers.CheckResult, _ []ocr2keepers.UpkeepPayload) error {
	eligible := 0
	for _, res := range results {
		if res.PipelineExecutionState == 0 && res.Eligible {
			eligible++
			p.resultsAdder.Add(res)
			if err := p.lggr.Collect(res.WorkID, uint64(res.Trigger.BlockNumber), telemetry.Queued); err != nil {
				p.lggr.Println(err.Error())
			}
		}
	}
	p.lggr.Printf("post-processing %d results, %d eligible\n", len(results), eligible)
	return nil
}
