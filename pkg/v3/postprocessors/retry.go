package postprocessors

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/smartcontractkit/chainlink-automation/pkg/v3/telemetry"
	ocr2keepers "github.com/smartcontractkit/chainlink-automation/pkg/v3/types"
)

func NewRetryablePostProcessor(q ocr2keepers.RetryQueue, logger *log.Logger) *retryablePostProcessor {
	return &retryablePostProcessor{
		logger: log.New(logger.Writer(), fmt.Sprintf("[%s | retryable-post-processor]", telemetry.ServiceName), telemetry.LogPkgStdFlags),
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
	retryable := 0
	for i, res := range results {
		if res.PipelineExecutionState != 0 && res.Retryable {
			e := p.q.Enqueue(ocr2keepers.RetryRecord{
				Payload: payloads[i],
				//Interval: res.RetryInterval,
			})
			if e == nil {
				retryable++
			}
			err = errors.Join(err, e)
		}
	}
	p.logger.Printf("post-processing %d results, %d retryable\n", len(results), retryable)
	return err
}
