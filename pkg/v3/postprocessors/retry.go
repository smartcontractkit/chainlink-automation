package postprocessors

import (
	"context"
	"errors"
	"log"

	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg/v3/types"
)

func NewRetryablePostProcessor(q ocr2keepers.RetryQueue, logger *log.Logger) *retryablePostProcessor {
	return &retryablePostProcessor{
		logger: logger,
		q:      q,
	}
}

type retryablePostProcessor struct {
	logger *log.Logger
	q      ocr2keepers.RetryQueue
}

var _ PostProcessor = (*retryablePostProcessor)(nil)

func (p *retryablePostProcessor) PostProcess(_ context.Context, results []ocr2keepers.CheckResult, payloads []ocr2keepers.UpkeepPayload) error {
	var err error

	for i, res := range results {
		if res.Retryable {
			err = errors.Join(err, p.q.Enqueue(payloads[i]))
		}
	}

	return err
}
