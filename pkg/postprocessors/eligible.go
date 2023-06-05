package postprocessors

import (
	"context"

	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg"
)

type checkResultAdder interface {
	Add(ocr2keepers.CheckResult)
}

type PostProcessor interface {
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
		if res.IsEligible() {
			p.resultsAdder.Add(res)
		}
	}

	return nil
}
