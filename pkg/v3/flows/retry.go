package flows

import (
	"context"
	"fmt"
	"log"
	"time"

	ocr2keepersv3 "github.com/smartcontractkit/ocr2keepers/pkg/v3"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/postprocessors"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/service"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/telemetry"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/tickers"
	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg/v3/types"
)

var (
	RetryBatchSize = 5
)

// log trigger flow is the happy path entry point for log triggered upkeeps
func newRetryFlow(
	preprocessors []ocr2keepersv3.PreProcessor[ocr2keepers.UpkeepPayload],
	rs ResultStore,
	rn ocr2keepersv3.Runner,
	retryQ ocr2keepers.RetryQueue,
	retryTickerInterval time.Duration,
	logger *log.Logger,
) service.Recoverable {
	// postprocessing is a combination of multiple smaller postprocessors
	post := postprocessors.NewCombinedPostprocessor(
		// create eligibility postprocessor with result store
		postprocessors.NewEligiblePostProcessor(rs, telemetry.WrapLogger(logger, "retry-eligible-postprocessor")),
		// create retry postprocessor
		postprocessors.NewRetryablePostProcessor(retryQ, telemetry.WrapLogger(logger, "retry-retryable-postprocessor")),
	)

	// create observer
	obs := ocr2keepersv3.NewRunnableObserver(
		preprocessors,
		post,
		rn,
		ObservationProcessLimit,
		log.New(logger.Writer(), fmt.Sprintf("[%s | retry-observer]", telemetry.ServiceName), telemetry.LogPkgStdFlags),
	)

	// create time ticker
	timeTick := tickers.NewTimeTicker[[]ocr2keepers.UpkeepPayload](retryTickerInterval, obs, func(ctx context.Context, _ time.Time) (tickers.Tick[[]ocr2keepers.UpkeepPayload], error) {
		return retryTick{logger: logger, q: retryQ, batchSize: RetryBatchSize}, nil
	}, log.New(logger.Writer(), fmt.Sprintf("[%s | retry-ticker]", telemetry.ServiceName), telemetry.LogPkgStdFlags))

	return timeTick
}

type retryTick struct {
	logger    *log.Logger
	q         ocr2keepers.RetryQueue
	batchSize int
}

func (t retryTick) Value(ctx context.Context) ([]ocr2keepers.UpkeepPayload, error) {
	if t.q == nil {
		return nil, nil
	}

	payloads, err := t.q.Dequeue(t.batchSize)
	if err != nil {
		return nil, fmt.Errorf("failed to dequeue from retry queue: %w", err)
	}
	t.logger.Printf("%d payloads returned by retry queue", len(payloads))

	return payloads, err
}
