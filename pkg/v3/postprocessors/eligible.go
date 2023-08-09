package postprocessors

import (
	"context"
	"log"

	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg/v3/types"
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
	PostProcess(context.Context, []ocr2keepers.CheckResult) error
}

type eligiblePostProcessor struct {
	lggr         *log.Logger
	resultsAdder checkResultAdder
}

func NewEligiblePostProcessor(resultsAdder checkResultAdder, logger *log.Logger) *eligiblePostProcessor {
	return &eligiblePostProcessor{
		lggr:         logger,
		resultsAdder: resultsAdder,
	}
}

func (p *eligiblePostProcessor) PostProcess(_ context.Context, results []ocr2keepers.CheckResult) error {
	eligible := 0
	for _, res := range results {
		if res.Eligible {
			eligible++
			p.resultsAdder.Add(res)
		}
	}
	p.lggr.Printf("[automation-ocr3/EligiblePostProcessor] post-processing %d results, %d eligible\n", len(results), eligible)
	return nil
}
