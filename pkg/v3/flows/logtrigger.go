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

var (
	ErrNotRetryable = fmt.Errorf("payload is not retryable")
)

//go:generate mockery --name Runner --structname MockRunner --srcpkg "github.com/smartcontractkit/ocr2keepers/pkg/v3/flows" --case underscore --filename runner.generated.go
type Runner interface {
	// CheckUpkeeps has an input of upkeeps with unknown state and an output of upkeeps with known state
	CheckUpkeeps(context.Context, ...ocr2keepers.UpkeepPayload) ([]ocr2keepers.CheckResult, error)
}

//go:generate mockery --name ResultStore --structname MockResultStore --srcpkg "github.com/smartcontractkit/ocr2keepers/pkg/v3/flows" --case underscore --filename resultstore.generated.go
type ResultStore interface {
	Add(...ocr2keepers.CheckResult)
}

// Retryer provides the ability to push retries to an observer
//
//go:generate mockery --name Retryer --structname MockRetryer --srcpkg "github.com/smartcontractkit/ocr2keepers/pkg/v3/flows" --case underscore --filename retryer.generated.go
type Retryer interface {
	// Retry provides an entry point for new retryable results
	Retry(ocr2keepers.CheckResult) error
}

const (
	LogCheckInterval        = 1 * time.Second
	RecoveryCheckInterval   = 1 * time.Minute
	ObservationProcessLimit = 5 * time.Second
)

// LogTriggerEligibility is a flow controller that surfaces eligible upkeeps
// with retry attempts.
type LogTriggerEligibility struct {
	builder ocr2keepers.PayloadBuilder
	mStore  store.MetadataStore
	logger  *log.Logger
}

// NewLogTriggerEligibility ...
func NewLogTriggerEligibility(
	coord ocr2keepersv3.PreProcessor[ocr2keepers.UpkeepPayload],
	rStore ResultStore,
	mStore store.MetadataStore,
	runner ocr2keepersv3.Runner,
	logProvider ocr2keepers.LogEventProvider,
	rp ocr2keepers.RecoverableProvider,
	builder ocr2keepers.PayloadBuilder,
	logInterval time.Duration,
	recoveryInterval time.Duration,
	retryQ ocr2keepers.RetryQueue,
	logger *log.Logger,
) (*LogTriggerEligibility, []service.Recoverable) {
	// all flows use the same preprocessor based on the coordinator
	// each flow can add preprocessors to this provided slice
	preprocessors := []ocr2keepersv3.PreProcessor[ocr2keepers.UpkeepPayload]{coord}

	// the recovery proposal flow is for nodes to surface payloads that should
	// be recovered. these values are passed to the network and the network
	// votes on the proposed values
	rcvProposal := newRecoveryProposalFlow(preprocessors, mStore, rp, recoveryInterval, logger)

	// the final recovery flow takes recoverable payloads merged with the latest
	// blocks and runs the pipeline for them. these values to run are derived
	// from node coordination and it can be assumed that all values should be
	// run.
	rcvFinal := newFinalRecoveryFlow(preprocessors, rStore, runner, retryQ, recoveryInterval, logger)

	// the log trigger flow is the happy path for log trigger payloads. all
	// retryables that are encountered in this flow are elevated to the retry
	// flow
	logTrigger := newLogTriggerFlow(preprocessors, rStore, runner, logProvider, logInterval, retryQ, logger)

	// all above flows run internal time-keeper services. each is essential for
	// running so the return is a slice of all above services as recoverables
	svcs := []service.Recoverable{
		rcvProposal,
		rcvFinal,
		logTrigger,
	}

	// the final return includes a struct that provides the ability for hooks
	// to pass data to internal flows
	return &LogTriggerEligibility{
		builder: builder,
		mStore:  mStore,
		logger:  logger,
	}, svcs
}

// ProcessOutcome functions as an observation pre-build hook to allow data from
// outcomes to feed inputs in the eligibility flow
func (flow *LogTriggerEligibility) ProcessOutcome(outcome ocr2keepersv3.AutomationOutcome) error {
	// TODO: Refactor into coordinatedProposals Flow
	/*networkProposals, err := outcome.RecoveryProposals()
	if err != nil {
		if errors.Is(err, ocr2keepersv3.ErrWrongDataType) {
			return err
		}

		flow.logger.Printf("%s", err)

		return nil
	}

	if len(networkProposals) == 0 {
		return nil
	}

	cachedProposals, err := store.RecoveryProposalCacheFromMetadata(flow.mStore)
	if err != nil {
		return err
	}

	// limit timeout to get all proposal data
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

	// merge block number and recoverables
	for _, proposal := range networkProposals {
		// remove from local metadata store
		cachedProposals.Delete(fmt.Sprintf("%v", proposal))

		payloads, err := flow.builder.BuildPayloads(ctx, proposal)
		if err != nil {
			flow.logger.Printf("error encountered when building payload")
			continue
		}
		if len(payloads) == 0 {
			flow.logger.Printf("did not get any results when building payload")
			continue
		}
		payload := payloads[0]

		// pass to recoverer
		if err := flow.recoverer.Retry(ocr2keepers.CheckResult{
			UpkeepID: payload.UpkeepID,
			Trigger:  payload.Trigger,
		}); err != nil {
			continue
		}
	}

	cachedProposals.ClearExpired()

	cancel()
	*/
	return nil
}

// log trigger flow is the happy path entry point for log triggered upkeeps
func newLogTriggerFlow(
	preprocessors []ocr2keepersv3.PreProcessor[ocr2keepers.UpkeepPayload],
	rs ResultStore,
	rn ocr2keepersv3.Runner,
	logProvider ocr2keepers.LogEventProvider,
	logInterval time.Duration,
	retryQ ocr2keepers.RetryQueue,
	logger *log.Logger,
) service.Recoverable {
	post := postprocessors.NewCombinedPostprocessor(
		postprocessors.NewEligiblePostProcessor(rs, telemetry.WrapLogger(logger, "log-trigger-eligible-postprocessor")),
		postprocessors.NewRetryablePostProcessor(retryQ, telemetry.WrapLogger(logger, "log-trigger-retryable-postprocessor")),
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
