package postprocessors

import (
	"context"

	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg"
)

// checkResultAdder is a general interface for a result store that accepts check results
type checkResultAdder interface {
	// Add inserts the provided check result in the store
	Add(ocr2keepers.CheckResult)
}

// PostProcessor is the general interface for a processing function after checking eligibility
// status
type PostProcessor interface {
	// PostProcess takes a slice of results where eligibility status is known
	PostProcess(context.Context, []ocr2keepers.CheckResult) error
}

type eligiblePostProcessor struct {
	resultsAdder checkResultAdder
}

func NewEligiblePostProcessor(resultsAdder checkResultAdder) *eligiblePostProcessor {
	return &eligiblePostProcessor{
		resultsAdder: resultsAdder,
	}
}

func (p *eligiblePostProcessor) PostProcess(_ context.Context, results []ocr2keepers.CheckResult) error {
	for _, res := range results {
		if res.Eligible {
			p.resultsAdder.Add(res)
		}
	}

	return nil
}
