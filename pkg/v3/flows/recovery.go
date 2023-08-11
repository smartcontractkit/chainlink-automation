package flows

import (
	"context"
	"fmt"
	"log"
	"time"

	ocr2keepersv3 "github.com/smartcontractkit/ocr2keepers/pkg/v3"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/postprocessors"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/service"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/store"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/telemetry"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/tickers"
	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg/v3/types"
)

func newFinalRecoveryFlow(
	preprocessors []ocr2keepersv3.PreProcessor[ocr2keepers.UpkeepPayload],
	rs ResultStore,
	rn ocr2keepersv3.Runner,
	retryQ ocr2keepers.RetryQueue,
	recoveryInterval time.Duration,
	logger *log.Logger,
) service.Recoverable {
	post := postprocessors.NewCombinedPostprocessor(
		postprocessors.NewEligiblePostProcessor(rs, telemetry.WrapLogger(logger, "recovery-final-eligible-postprocessor")),
		postprocessors.NewRetryablePostProcessor(retryQ, telemetry.WrapLogger(logger, "recovery-final-retryable-postprocessor")),
	)

	// create observer that only pushes results to result store. everything at
	// this point can be dropped. this process is only responsible for running
	// recovery proposals that originate from network agreements
	recoveryObserver := ocr2keepersv3.NewRunnableObserver(
		preprocessors,
		post,
		rn,
		ObservationProcessLimit,
		log.New(logger.Writer(), fmt.Sprintf("[%s | recovery-final-observer]", telemetry.ServiceName), telemetry.LogPkgStdFlags),
	)

	// create schedule ticker to manage retry interval
	ticker := tickers.NewBasicTicker[ocr2keepers.UpkeepPayload](
		recoveryInterval,
		recoveryObserver,
		log.New(logger.Writer(), fmt.Sprintf("[%s | recovery-final-ticker]", telemetry.ServiceName), telemetry.LogPkgStdFlags),
	)

	return ticker
}

func newRecoveryProposalFlow(
	preprocessors []ocr2keepersv3.PreProcessor[ocr2keepers.UpkeepPayload],
	ms store.MetadataStore,
	rp ocr2keepers.RecoverableProvider,
	recoveryInterval time.Duration,
	logger *log.Logger,
	configFuncs ...tickers.ScheduleTickerConfigFunc,
) service.Recoverable {
	// items come into the recovery path from multiple sources
	// 1. [done] from the log provider as UpkeepPayload
	// 2. [done] from retry ticker as CheckResult
	// 3. [done] from primary flow as CheckResult if retry fails
	// 4. [todo] from timeouts of the result store
	// TODO: add preprocessor to check that recoverable is already in metadata

	// the recovery observer doesn't do any processing on the identifiers
	// so this function is just a pass-through
	// TODO: align
	f := func(_ context.Context, payloads ...ocr2keepers.UpkeepPayload) ([]ocr2keepers.CheckResult, error) {
		results := make([]ocr2keepers.CheckResult, len(payloads))
		for i, id := range payloads {
			results[i] = ocr2keepers.CheckResult{
				WorkID:   id.WorkID,
				UpkeepID: id.UpkeepID,
				Trigger:  id.Trigger,
			}
		}
		return results, nil
	}

	// the recovery observer is just a pass-through to the metadata store
	// add postprocessor for metatdata store
	// TODO: align with new metadata store API
	post := postprocessors.NewAddPayloadToMetadataStorePostprocessor(ms)

	recoveryObserver := ocr2keepersv3.NewGenericObserver[ocr2keepers.UpkeepPayload, ocr2keepers.CheckResult](
		preprocessors,
		post,
		f,
		ObservationProcessLimit,
		log.New(logger.Writer(), fmt.Sprintf("[%s | recovery-proposal-observer]", telemetry.ServiceName), telemetry.LogPkgStdFlags),
	)

	// create a schedule ticker that pulls recoverable items from an outside
	// source and provides point for recoverables to be pushed to the ticker
	// TODO: time ticker, fetches from RecoverableProvider
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

	return ticker
}
