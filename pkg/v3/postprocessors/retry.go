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
	recoverer checkResultRetryer
}

func (p *retryPostProcessor) PostProcess(_ context.Context, results []ocr2keepers.CheckResult) error {
	var err error

	for _, res := range results {
		if res.Retryable {
			err = errors.Join(err, p.attemptRetry(res))
		}
	}

	return err
}

func (p *retryPostProcessor) attemptRetry(res ocr2keepers.CheckResult) error {
	err := p.retryer.Retry(res)
	if err == nil {
		return nil
	}

	if errors.Is(err, tickers.ErrSendDurationExceeded) {
		if err := p.recoverer.Retry(res); err != nil {
			return err
		}

		return nil
	}

	return err
}

func NewRetryPostProcessor(retryer checkResultRetryer, recoverer checkResultRetryer) *retryPostProcessor {
	return &retryPostProcessor{
		retryer:   retryer,
		recoverer: recoverer,
	}
}
