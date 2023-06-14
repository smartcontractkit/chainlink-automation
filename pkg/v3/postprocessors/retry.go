package postprocessors

import (
	"context"

	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg"
)

type checkResultRetryer interface {
	Retry(ocr2keepers.CheckResult) error
}

type retryPostProcessor struct {
	retryer checkResultRetryer
}

func (p *retryPostProcessor) PostProcess(_ context.Context, results []ocr2keepers.CheckResult) error {
	for _, res := range results {
		if res.Retryable {
			if err := p.retryer.Retry(res); err != nil {
				return err
			}
		}
	}
	return nil
}

func NewRetryPostProcessor(retryer checkResultRetryer) *retryPostProcessor {
	return &retryPostProcessor{
		retryer: retryer,
	}
}
