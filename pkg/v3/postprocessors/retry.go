package postprocessors

import (
	"context"
	"errors"

	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/tickers"
)

type checkResultRetryer interface {
	Retry(ocr2keepers.CheckResult) error
}

type retryPostProcessor struct {
	retryer   checkResultRetryer
	recoverer checkResultRecoverer
}

func (p *retryPostProcessor) PostProcess(_ context.Context, results []ocr2keepers.CheckResult) error {
	for _, res := range results {
		if res.Retryable {
			if err := p.retryer.Retry(res); err != nil {
				// TODO Aggregate the errors, don't short circuit
				if errors.Is(err, tickers.ErrRetryDurationExceeded) {
					res.Recoverable = true
					res.Retryable = false
					p.recoverer.Recover(res)
				}
				return err
			}
		}
	}
	return nil
}

func NewRetryPostProcessor(retryer checkResultRetryer, recoverer checkResultRecoverer) *retryPostProcessor {
	return &retryPostProcessor{
		retryer: retryer,
	}
}
