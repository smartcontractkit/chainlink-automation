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

func newFinalRecoveryFlow(
	preprocessors []ocr2keepersv3.PreProcessor[ocr2keepers.UpkeepPayload],
	rs ResultStore,
	rn ocr2keepersv3.Runner,
	recoverer Retryer,
	recoveryInterval time.Duration,
	logger *log.Logger,
) (service.Recoverable, Retryer) {
	// postprocessing is a combination of multiple smaller postprocessors
	post := postprocessors.NewCombinedPostprocessor(
		// create eligibility postprocessor with result store
		postprocessors.NewEligiblePostProcessor(rs, log.New(logger.Writer(), fmt.Sprintf("[%s | recovery-eligible-postprocessor]", telemetry.ServiceName), telemetry.LogPkgStdFlags)),
		// create retry postprocessor
		postprocessors.NewRetryPostProcessor(nil, recoverer),
	)

	// create observer that only pushes results to result store. everything at
	// this point can be dropped. this process is only responsible for running
	// recovery proposals that originate from network agreements
	recoveryObserver := ocr2keepersv3.NewRunnableObserver(
		preprocessors,
		post,
		rn,
		ObservationProcessLimit,
		log.New(logger.Writer(), fmt.Sprintf("[%s | recovery-observer]", telemetry.ServiceName), telemetry.LogPkgStdFlags),
	)

	// create schedule ticker to manage retry interval
	ticker := tickers.NewBasicTicker[ocr2keepers.UpkeepPayload](
		recoveryInterval,
		recoveryObserver,
		log.New(logger.Writer(), fmt.Sprintf("[%s | log-trigger-final-recovery]", telemetry.ServiceName), telemetry.LogPkgStdFlags),
	)

	// wrap schedule ticker as a Retryer
	// this provides a common interface for processors and hooks
	retryer := &basicRetryer{ticker: ticker}

	return ticker, retryer
}

func newRecoveryProposalFlow(
	preprocessors []ocr2keepersv3.PreProcessor[ocr2keepers.UpkeepPayload],
	ms MetadataStore,
	rp ocr2keepers.RecoverableProvider,
	recoveryInterval time.Duration,
	logger *log.Logger,
	configFuncs ...tickers.ScheduleTickerConfigFunc,
) (service.Recoverable, Retryer) {
	// items come into the recovery path from multiple sources
	// 1. [done] from the log provider as UpkeepPayload
	// 2. [done] from retry ticker as CheckResult
	// 3. [done] from primary flow as CheckResult if retry fails
	// 4. [todo] from timeouts of the result store
	// TODO: add preprocessor to check that recoverable is already in metadata

	// the recovery observer doesn't do any processing on the identifiers
	// so this function is just a pass-through
	f := func(_ context.Context, ids ...ocr2keepers.UpkeepPayload) ([]ocr2keepers.UpkeepPayload, error) {
		return ids, nil
	}

	// the recovery observer is just a pass-through to the metadata store
	// add postprocessor for metatdata store
	post := postprocessors.NewAddPayloadToMetadataStorePostprocessor(ms)

	recoveryObserver := ocr2keepersv3.NewGenericObserver[ocr2keepers.UpkeepPayload, ocr2keepers.UpkeepPayload](
		preprocessors,
		post,
		f,
		ObservationProcessLimit,
		log.New(logger.Writer(), fmt.Sprintf("[%s | recovery-proposal-observer]", telemetry.ServiceName), telemetry.LogPkgStdFlags),
	)

	// create a schedule ticker that pulls recoverable items from an outside
	// source and provides point for recoverables to be pushed to the ticker
	ticker := tickers.NewScheduleTicker[ocr2keepers.UpkeepPayload](
		recoveryInterval,
		recoveryObserver,
		func(f func(string, ocr2keepers.UpkeepPayload) error) error {
			// TODO: Pass in parent context to this function
			ctx := context.Background()
			// pull payloads from RecoverableProvider
			recovers, err := rp.GetRecoveryProposals(ctx)
			if err != nil {
				return err
			}

			for _, rec := range recovers {
				if err := f(rec.WorkID, rec); err != nil {
					return err
				}
			}

			return nil
		},
		log.New(logger.Writer(), fmt.Sprintf("[%s | log-trigger-recovery-proposal]", telemetry.ServiceName), telemetry.LogPkgStdFlags),
		configFuncs...,
	)

	// wrap schedule ticker as a Retryer
	// this provides a common interface for processors and hooks
	retryer := &scheduledRetryer{scheduler: ticker}

	return ticker, retryer
}
