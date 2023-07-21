package flows

import (
	"context"
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

func newRecoveryProposalFlow(preprocessors []ocr2keepersv3.PreProcessor[ocr2keepers.UpkeepPayload], ms MetadataStore, rp RecoverableProvider, recoveryInterval time.Duration, logger *log.Logger, configFuncs ...tickers.ScheduleTickerConfigFunc) (service.Recoverable, Retryer) {
	// items come into the recovery path from multiple sources
	// 1. [done] from the log provider as UpkeepPayload
	// 2. [done] from retry ticker as CheckResult
	// 3. [done] from primary flow as CheckResult if retry fails
	// 4. [todo] from timeouts of the result store

	// the recovery observer doesn't do any processing on the identifiers
	// so this function is just a pass-through
	f := func(_ context.Context, ids ...ocr2keepers.UpkeepPayload) ([]ocr2keepers.UpkeepPayload, error) {
		return ids, nil
	}

	// the recovery observer is just a pass-through to the metadata store
	// add postprocessor for metatdata store
	post := postprocessors.NewAddPayloadToMetadataStorePostprocessor(ms)

	recoveryObserver := ocr2keepersv3.NewGenericObserver[ocr2keepers.UpkeepPayload, ocr2keepers.UpkeepPayload](preprocessors, post, f, ObservationProcessLimit)

	// create a schedule ticker that pulls recoverable items from an outside
	// source and provides point for recoverables to be pushed to the ticker
	ticker := tickers.NewScheduleTicker[ocr2keepers.UpkeepPayload](
		recoveryInterval,
		recoveryObserver,
		func(f func(string, ocr2keepers.UpkeepPayload) error) error {
			// pull payloads from RecoverableProvider
			recovers, err := rp.GetRecoverables()
			if err != nil {
				return err
			}

			for _, rec := range recovers {
				if err := f(rec.ID, rec); err != nil {
					return err
				}
			}

			return nil
		},
		log.New(logger.Writer(), fmt.Sprintf("[%s | log-trigger-primary]", telemetry.ServiceName), telemetry.LogPkgStdFlags),
		configFuncs...,
	)

	// wrap schedule ticker as a Retryer
	// this provides a common interface for processors and hooks
	retryer := &scheduledRetryer{scheduler: ticker}

	return ticker, retryer
}
