package postprocessors

import (
	"context"
)

type CheckResult struct {
	Retryable bool
}

type checkResultRetryer interface {
	Retry(CheckResult)
}

type PostProcessor interface {
	PostProcess(context.Context, []CheckResult) error
}

type retryPostProcessor struct {
	retryer checkResultRetryer
}

func (p *retryPostProcessor) PostProcess(_ context.Context, results []CheckResult) error {
	for _, res := range results {
		if res.Retryable {
			p.retryer.Retry(res)
		}
	}
	return nil
}

func NewRetryPostProcessor(retryer checkResultRetryer) *retryPostProcessor {
	return &retryPostProcessor{
		retryer: retryer,
	}
}
