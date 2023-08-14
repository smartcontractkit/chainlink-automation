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
	ErrNotRetryable = fmt.Errorf("payload is not retryable")
)

const (
	// This is the ticker interval for log trigger flow
	LogCheckInterval = 1 * time.Second
	// Limit for processing a whole observer flow given a payload. The main component of this
	// is the checkPipeline which involves some RPC, DB and Mercury calls, this is limited
	// to 20 seconds for now
	ObservationProcessLimit = 20 * time.Second
)

// log trigger flow is the happy path entry point for log triggered upkeeps
func newLogTriggerFlow(
	preprocessors []ocr2keepersv3.PreProcessor[ocr2keepers.UpkeepPayload],
	rs ocr2keepers.ResultStore,
	rn ocr2keepersv3.Runner,
	logProvider ocr2keepers.LogEventProvider,
	logInterval time.Duration,
	retryQ ocr2keepers.RetryQueue,
	stateUpdater ocr2keepers.UpkeepStateUpdater,
	logger *log.Logger,
) service.Recoverable {
	post := postprocessors.NewCombinedPostprocessor(
		postprocessors.NewEligiblePostProcessor(rs, telemetry.WrapLogger(logger, "log-trigger-eligible-postprocessor")),
		postprocessors.NewRetryablePostProcessor(retryQ, telemetry.WrapLogger(logger, "log-trigger-retryable-postprocessor")),
		postprocessors.NewIneligiblePostProcessor(stateUpdater, telemetry.WrapLogger(logger, "retry-ineligible-postprocessor")),
	)

	obs := ocr2keepersv3.NewRunnableObserver(
		preprocessors,
		post,
		rn,
		ObservationProcessLimit,
		log.New(logger.Writer(), fmt.Sprintf("[%s | log-trigger-observer]", telemetry.ServiceName), telemetry.LogPkgStdFlags),
	)

	timeTick := tickers.NewTimeTicker[[]ocr2keepers.UpkeepPayload](logInterval, obs, func(ctx context.Context, _ time.Time) (tickers.Tick[[]ocr2keepers.UpkeepPayload], error) {
		return logTick{logger: logger, logProvider: logProvider}, nil
	}, log.New(logger.Writer(), fmt.Sprintf("[%s | log-trigger-ticker]", telemetry.ServiceName), telemetry.LogPkgStdFlags))

	return timeTick
}

type logTick struct {
	logProvider ocr2keepers.LogEventProvider
	logger      *log.Logger
}

func (et logTick) Value(ctx context.Context) ([]ocr2keepers.UpkeepPayload, error) {
	if et.logProvider == nil {
		return nil, nil
	}

	logs, err := et.logProvider.GetLatestPayloads(ctx)

	et.logger.Printf("%d logs returned by log provider", len(logs))

	return logs, err
}
