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
	proposalQ ocr2keepers.ProposalQueue,
	builder ocr2keepers.PayloadBuilder,
	stateUpdater ocr2keepers.UpkeepStateUpdater,
	logger *log.Logger,
) service.Recoverable {
	post := postprocessors.NewCombinedPostprocessor(
		postprocessors.NewEligiblePostProcessor(rs, telemetry.WrapLogger(logger, "recovery-final-eligible-postprocessor")),
		postprocessors.NewRetryablePostProcessor(retryQ, telemetry.WrapLogger(logger, "recovery-final-retryable-postprocessor")),
		postprocessors.NewIneligiblePostProcessor(stateUpdater, telemetry.WrapLogger(logger, "retry-ineligible-postprocessor")),
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

	ticker := tickers.NewTimeTicker[[]ocr2keepers.UpkeepPayload](recoveryInterval, recoveryObserver, func(ctx context.Context, _ time.Time) (tickers.Tick[[]ocr2keepers.UpkeepPayload], error) {
		return coordinatedProposalsTick{
			logger:    logger,
			builder:   builder,
			q:         proposalQ,
			utype:     ocr2keepers.LogTrigger,
			batchSize: RetryBatchSize,
		}, nil
	}, log.New(logger.Writer(), fmt.Sprintf("[%s | recovery-final-ticker]", telemetry.ServiceName), telemetry.LogPkgStdFlags))

	return ticker
}

// coordinatedProposalsTick is used to push proposals from the proposal queue to some observer
type coordinatedProposalsTick struct {
	logger    *log.Logger
	builder   ocr2keepers.PayloadBuilder
	q         ocr2keepers.ProposalQueue
	utype     ocr2keepers.UpkeepType
	batchSize int
}

func (t coordinatedProposalsTick) Value(ctx context.Context) ([]ocr2keepers.UpkeepPayload, error) {
	if t.q == nil {
		return nil, nil
	}

	proposals, err := t.q.Dequeue(t.utype, t.batchSize)
	if err != nil {
		return nil, fmt.Errorf("failed to dequeue from retry queue: %w", err)
	}
	t.logger.Printf("%d proposals returned from queue", len(proposals))

	return t.builder.BuildPayloads(ctx, proposals...)
}

func newRecoveryProposalFlow(
	preprocessors []ocr2keepersv3.PreProcessor[ocr2keepers.UpkeepPayload],
	ms store.MetadataStore,
	rp ocr2keepers.RecoverableProvider,
	recoveryInterval time.Duration,
	typeGetter ocr2keepers.UpkeepTypeGetter,
	logger *log.Logger,
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
	post := postprocessors.NewAddPayloadToMetadataStorePostprocessor(ms, typeGetter)

	obs := ocr2keepersv3.NewGenericObserver[ocr2keepers.UpkeepPayload, ocr2keepers.CheckResult](
		preprocessors,
		post,
		f,
		ObservationProcessLimit,
		log.New(logger.Writer(), fmt.Sprintf("[%s | recovery-proposal-observer]", telemetry.ServiceName), telemetry.LogPkgStdFlags),
	)

	timeTick := tickers.NewTimeTicker[[]ocr2keepers.UpkeepPayload](recoveryInterval, obs, func(ctx context.Context, _ time.Time) (tickers.Tick[[]ocr2keepers.UpkeepPayload], error) {
		return logRecoveryTick{logger: logger, logRecoverer: rp}, nil
	}, log.New(logger.Writer(), fmt.Sprintf("[%s | recovery-proposal-ticker]", telemetry.ServiceName), telemetry.LogPkgStdFlags))

	return timeTick
}

type logRecoveryTick struct {
	logRecoverer ocr2keepers.RecoverableProvider
	logger       *log.Logger
}

func (et logRecoveryTick) Value(ctx context.Context) ([]ocr2keepers.UpkeepPayload, error) {
	if et.logRecoverer == nil {
		return nil, nil
	}

	logs, err := et.logRecoverer.GetRecoveryProposals(ctx)

	et.logger.Printf("%d logs returned by log recoverer", len(logs))

	return logs, err
}
