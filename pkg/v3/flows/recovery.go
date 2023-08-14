package flows

import (
	"context"
	"fmt"
	"log"
	"time"

	ocr2keepersv3 "github.com/smartcontractkit/ocr2keepers/pkg/v3"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/postprocessors"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/preprocessors"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/service"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/telemetry"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/tickers"
	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg/v3/types"
)

const (
	// These are the maximum number of log upkeeps dequeued on every tick from proposal queue in FinalRecoveryFlow
	// This is kept same as OutcomeSurfacedProposalsLimit as those many can get enqueued by plugin in every round
	FinalRecoveryBatchSize = 50
	// This is the ticker interval for final recovery flow
	RecoveryCheckInterval = 1 * time.Minute
)

func newFinalRecoveryFlow(
	preprocessors []ocr2keepersv3.PreProcessor[ocr2keepers.UpkeepPayload],
	resultStore ocr2keepers.ResultStore,
	runner ocr2keepersv3.Runner,
	retryQ ocr2keepers.RetryQueue,
	recoveryFinalizationInterval time.Duration,
	proposalQ ocr2keepers.ProposalQueue,
	builder ocr2keepers.PayloadBuilder,
	stateUpdater ocr2keepers.UpkeepStateUpdater,
	logger *log.Logger,
) service.Recoverable {
	post := postprocessors.NewCombinedPostprocessor(
		postprocessors.NewEligiblePostProcessor(resultStore, telemetry.WrapLogger(logger, "recovery-final-eligible-postprocessor")),
		postprocessors.NewRetryablePostProcessor(retryQ, telemetry.WrapLogger(logger, "recovery-final-retryable-postprocessor")),
		postprocessors.NewIneligiblePostProcessor(stateUpdater, telemetry.WrapLogger(logger, "retry-ineligible-postprocessor")),
	)
	// create observer that only pushes results to result stores. everything at
	// this point can be dropped. this process is only responsible for running
	// recovery proposals that originate from network agreements
	recoveryObserver := ocr2keepersv3.NewRunnableObserver(
		preprocessors,
		post,
		runner,
		ObservationProcessLimit,
		log.New(logger.Writer(), fmt.Sprintf("[%s | recovery-final-observer]", telemetry.ServiceName), telemetry.LogPkgStdFlags),
	)

	ticker := tickers.NewTimeTicker[[]ocr2keepers.UpkeepPayload](recoveryFinalizationInterval, recoveryObserver, func(ctx context.Context, _ time.Time) (tickers.Tick[[]ocr2keepers.UpkeepPayload], error) {
		return coordinatedProposalsTick{
			logger:    logger,
			builder:   builder,
			q:         proposalQ,
			utype:     ocr2keepers.LogTrigger,
			batchSize: FinalRecoveryBatchSize,
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
	preProcessors []ocr2keepersv3.PreProcessor[ocr2keepers.UpkeepPayload],
	runner ocr2keepersv3.Runner,
	metadataStore ocr2keepers.MetadataStore,
	recoverableProvider ocr2keepers.RecoverableProvider,
	recoveryInterval time.Duration,
	logger *log.Logger,
) service.Recoverable {
	preProcessors = append(preProcessors, preprocessors.NewProposalFilterer(metadataStore, ocr2keepers.LogTrigger))
	postprocessors := postprocessors.NewAddProposalToMetadataStorePostprocessor(metadataStore)

	observer := ocr2keepersv3.NewRunnableObserver(
		preProcessors,
		postprocessors,
		runner,
		ObservationProcessLimit,
		log.New(logger.Writer(), fmt.Sprintf("[%s | recovery-proposal-observer]", telemetry.ServiceName), telemetry.LogPkgStdFlags),
	)

	return tickers.NewTimeTicker[[]ocr2keepers.UpkeepPayload](recoveryInterval, observer, func(ctx context.Context, _ time.Time) (tickers.Tick[[]ocr2keepers.UpkeepPayload], error) {
		return logRecoveryTick{logger: logger, logRecoverer: recoverableProvider}, nil
	}, log.New(logger.Writer(), fmt.Sprintf("[%s | recovery-proposal-ticker]", telemetry.ServiceName), telemetry.LogPkgStdFlags))
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
