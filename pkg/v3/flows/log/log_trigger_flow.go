package log

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
	"github.com/smartcontractkit/chainlink-automation/pkg/v3/types"
	common "github.com/smartcontractkit/chainlink-common/pkg/types/automation"
)

var (
	ErrNotRetryable = fmt.Errorf("payload is not retryable")
)

const (
	// This is the ticker interval for log trigger flow
	logCheckInterval = 1 * time.Second
	// Limit for processing a whole observer flow given a payload. The main component of this
	// is the checkPipeline which involves some RPC, DB and Mercury calls, this is limited
	// to 20 seconds for now
	observationProcessLimit = 20 * time.Second
)

// log trigger flow is the happy path entry point for log triggered upkeeps
func NewLogTriggerFlow(
	preprocessors []ocr2keepersv3.PreProcessor[common.UpkeepPayload],
	rs types.ResultStore,
	rn ocr2keepersv3.Runner,
	logProvider common.LogEventProvider,
	retryQ types.RetryQueue,
	stateUpdater common.UpkeepStateUpdater,
	logger *log.Logger,
) service.Recoverable {
	post := postprocessors.NewCombinedPostprocessor(
		postprocessors.NewEligiblePostProcessor(rs, telemetry.WrapLogger(logger, "log-trigger-eligible-postprocessor")),
		postprocessors.NewRetryablePostProcessor(retryQ, telemetry.WrapLogger(logger, "log-trigger-retryable-postprocessor")),
		postprocessors.NewIneligiblePostProcessor(stateUpdater, telemetry.WrapLogger(logger, "retry-ineligible-postprocessor")),
	)

	observer := ocr2keepersv3.NewRunnableObserver(
		preprocessors,
		post,
		rn,
		observationProcessLimit,
		log.New(logger.Writer(), fmt.Sprintf("[%s | log-trigger-observer]", telemetry.ServiceName), telemetry.LogPkgStdFlags),
	)

	getterFn := func(ctx context.Context, _ time.Time) (tickers.Tick[[]common.UpkeepPayload], error) {
		return logTick{logger: logger, logProvider: logProvider}, nil
	}

	lgrPrefix := fmt.Sprintf("[%s | log-trigger-ticker]", telemetry.ServiceName)
	lggr := log.New(logger.Writer(), lgrPrefix, telemetry.LogPkgStdFlags)

	timeTick := tickers.NewTimeTicker[[]common.UpkeepPayload](logCheckInterval, observer, getterFn, lggr)

	return timeTick
}

type logTick struct {
	logProvider common.LogEventProvider
	logger      *log.Logger
}

func (t logTick) Value(ctx context.Context) ([]common.UpkeepPayload, error) {
	if t.logProvider == nil {
		return nil, nil
	}

	logs, err := t.logProvider.GetLatestPayloads(ctx)

	t.logger.Printf("%d logs returned by log provider", len(logs))

	return logs, err
}
