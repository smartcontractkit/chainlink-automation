package flows

import (
	"fmt"
	"log"
	"time"

	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg"
	ocr2keepersv3 "github.com/smartcontractkit/ocr2keepers/pkg/v3"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/postprocessors"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/service"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/telemetry"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/tickers"
)

// Should be same as normal observer, just without the ticker
func newRetryFlow(
	preprocessors []ocr2keepersv3.PreProcessor[ocr2keepers.UpkeepPayload],
	rs ResultStore,
	rn Runner,
	recoverer Retryer,
	retryInterval time.Duration,
	logger *log.Logger,
	configFuncs ...tickers.ScheduleTickerConfigFunc,
) (service.Recoverable, Retryer) {
	// create observer
	// leave postprocessor empty to start with
	retryObserver := ocr2keepersv3.NewRunnableObserver(
		preprocessors,
		nil,
		rn,
		ObservationProcessLimit,
		log.New(logger.Writer(), fmt.Sprintf("[%s | retry-observer]", telemetry.ServiceName), telemetry.LogPkgStdFlags),
	)

	// create schedule ticker to manage retry interval
	ticker := tickers.NewScheduleTicker[ocr2keepers.UpkeepPayload](
		retryInterval,
		retryObserver,
		func(func(string, ocr2keepers.UpkeepPayload) error) error {
			// this schedule ticker doesn't pull data from anywhere
			return nil
		},
		log.New(logger.Writer(), fmt.Sprintf("[%s | log-trigger-retry]", telemetry.ServiceName), telemetry.LogPkgStdFlags),
		configFuncs...,
	)

	// wrap schedule ticker as a Retryer
	// this provides a common interface for processors and hooks
	retryer := &scheduledRetryer{scheduler: ticker}

	// postprocessing is a combination of multiple smaller postprocessors
	post := postprocessors.NewCombinedPostprocessor(
		// create eligibility postprocessor with result store
		postprocessors.NewEligiblePostProcessor(rs, log.New(logger.Writer(), fmt.Sprintf("[%s | retry-eligible-postprocessor]", telemetry.ServiceName), telemetry.LogPkgStdFlags)),
		// create retry postprocessor
		postprocessors.NewRetryPostProcessor(retryer, recoverer),
	)

	retryObserver.SetPostProcessor(post)

	// return retry ticker as a recoverable and retryer
	return ticker, retryer
}
