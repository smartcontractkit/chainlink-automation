package flows

import (
	"context"
	"fmt"
	"log"
	"time"

	ocr2keepersv3 "github.com/smartcontractkit/chainlink-automation/pkg/v3"
	"github.com/smartcontractkit/chainlink-automation/pkg/v3/postprocessors"
	"github.com/smartcontractkit/chainlink-automation/pkg/v3/service"
	"github.com/smartcontractkit/chainlink-automation/pkg/v3/telemetry"
	"github.com/smartcontractkit/chainlink-automation/pkg/v3/tickers"
	ocr2keepers "github.com/smartcontractkit/chainlink-automation/pkg/v3/types"
)

const (
	// These are the max number of payloads dequeued on every tick from the retry queue in the retry flow
	RetryBatchSize = 10
	// This is the ticker interval for retry flow
	RetryCheckInterval = 5 * time.Second
)

func NewRetryFlow(
	coord ocr2keepersv3.PreProcessor[ocr2keepers.UpkeepPayload],
	resultStore ocr2keepers.ResultStore,
	runner ocr2keepersv3.Runner,
	retryQ ocr2keepers.RetryQueue,
	retryTickerInterval time.Duration,
	stateUpdater ocr2keepers.UpkeepStateUpdater,
	logger *telemetry.Logger,
) service.Recoverable {
	preprocessors := []ocr2keepersv3.PreProcessor[ocr2keepers.UpkeepPayload]{coord}
	post := postprocessors.NewCombinedPostprocessor(
		postprocessors.NewEligiblePostProcessor(resultStore, telemetry.WrapTelemetryLogger(logger, "retry-eligible-postprocessor")),
		postprocessors.NewRetryablePostProcessor(retryQ, telemetry.WrapTelemetryLogger(logger, "retry-retryable-postprocessor")),
		postprocessors.NewIneligiblePostProcessor(stateUpdater, telemetry.WrapTelemetryLogger(logger, "retry-ineligible-postprocessor")),
	)

	obs := ocr2keepersv3.NewRunnableObserver(
		preprocessors,
		post,
		runner,
		ObservationProcessLimit,
		log.New(logger.Writer(), fmt.Sprintf("[%s | retry-observer]", telemetry.ServiceName), telemetry.LogPkgStdFlags),
	)

	timeTick := tickers.NewTimeTicker[[]ocr2keepers.UpkeepPayload](retryTickerInterval, obs, func(ctx context.Context, _ time.Time) (tickers.Tick[[]ocr2keepers.UpkeepPayload], error) {
		return retryTick{logger: logger, q: retryQ, batchSize: RetryBatchSize}, nil
	}, log.New(logger.Writer(), fmt.Sprintf("[%s | retry-ticker]", telemetry.ServiceName), telemetry.LogPkgStdFlags))

	return timeTick
}

type retryTick struct {
	logger    *telemetry.Logger
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
