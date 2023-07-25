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
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/store"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/telemetry"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/tickers"
)

var (
	ErrNotRetryable = fmt.Errorf("payload is not retryable")
)

//go:generate mockery --name Runner --structname MockRunner --srcpkg "github.com/smartcontractkit/ocr2keepers/pkg/v3/flows" --case underscore --filename runner.generated.go
type Runner interface {
	// CheckUpkeeps has an input of upkeeps with unknown state and an output of upkeeps with known state
	CheckUpkeeps(context.Context, ...ocr2keepers.UpkeepPayload) ([]ocr2keepers.CheckResult, error)
}

// TODO cleanup, this one is not used, ocr2keepersv3.PreProcessor is used instead
//
//go:generate mockery --name PreProcessor --structname MockPreProcessor --srcpkg "github.com/smartcontractkit/ocr2keepers/pkg/v3/flows" --case underscore --filename preprocessor.generated.go
type PreProcessor interface {
	// PreProcess takes a slice of payloads and returns a new slice
	PreProcess(context.Context, []ocr2keepers.UpkeepPayload) ([]ocr2keepers.UpkeepPayload, error)
}

//go:generate mockery --name ResultStore --structname MockResultStore --srcpkg "github.com/smartcontractkit/ocr2keepers/pkg/v3/flows" --case underscore --filename resultstore.generated.go
type ResultStore interface {
	Add(...ocr2keepers.CheckResult)
}

//go:generate mockery --name MetadataStore --structname MockMetadataStore --srcpkg "github.com/smartcontractkit/ocr2keepers/pkg/v3/flows" --case underscore --filename metadatastore.generated.go
type MetadataStore interface {
	Set(store.MetadataKey, interface{})
}

// Retryer provides the ability to push retries to an observer
//
//go:generate mockery --name Retryer --structname MockRetryer --srcpkg "github.com/smartcontractkit/ocr2keepers/pkg/v3/flows" --case underscore --filename retryer.generated.go
type Retryer interface {
	// Retry provides an entry point for new retryable results
	Retry(ocr2keepers.CheckResult) error
}

//go:generate mockery --name LogEventProvider --structname MockLogEventProvider --srcpkg "github.com/smartcontractkit/ocr2keepers/pkg/v3/flows" --case underscore --filename logeventprovider.generated.go
type LogEventProvider interface {
	// GetLogs returns the latest logs
	GetLogs(context.Context) ([]ocr2keepers.UpkeepPayload, error)
}

//go:generate mockery --name RecoverableProvider --structname MockRecoverableProvider --srcpkg "github.com/smartcontractkit/ocr2keepers/pkg/v3/flows" --case underscore --filename recoverableprovider.generated.go
type RecoverableProvider interface {
	GetRecoverables() ([]ocr2keepers.UpkeepPayload, error)
}

//go:generate mockery --name PayloadBuilder --structname MockPayloadBuilder --srcpkg "github.com/smartcontractkit/ocr2keepers/pkg/v3/flows" --case underscore --filename payloadbuilder.generated.go
type PayloadBuilder interface {
	BuildPayload(context.Context, ocr2keepers.CoordinatedProposal) (ocr2keepers.UpkeepPayload, error)
}

const (
	LogCheckInterval        = 1 * time.Second
	RecoveryCheckInterval   = 1 * time.Minute
	ObservationProcessLimit = 5 * time.Second
)

var (
	ErrWrongDataType     = fmt.Errorf("wrong data type")
	ErrBlockNotAvailable = fmt.Errorf("recovery proposals are available but a coordinated block is not")
)

// LogTriggerEligibility is a flow controller that surfaces eligible upkeeps
// with retry attempts.
type LogTriggerEligibility struct {
	builder   PayloadBuilder
	recoverer Retryer
	logger    *log.Logger
}

// NewLogTriggerEligibility ...
func NewLogTriggerEligibility(
	coord PreProcessor,
	rStore ResultStore,
	mStore MetadataStore,
	runner Runner,
	logProvider LogEventProvider,
	rp RecoverableProvider,
	builder PayloadBuilder,
	logInterval time.Duration,
	recoveryInterval time.Duration,
	logger *log.Logger,
	retryConfigs []tickers.ScheduleTickerConfigFunc,
	recoverConfigs []tickers.ScheduleTickerConfigFunc,
) (*LogTriggerEligibility, []service.Recoverable) {
	// all flows use the same preprocessor based on the coordinator
	// each flow can add preprocessors to this provided slice
	preprocessors := []ocr2keepersv3.PreProcessor[ocr2keepers.UpkeepPayload]{coord}

	// the final recovery flow takes recoverable payloads merged with the latest
	// blocks and runs the pipeline for them. these values to run are derived
	// from node coordination and it can be assumed that all values should be
	// run.
	svc0, recoverer := newFinalRecoveryFlow(preprocessors, rStore, runner, recoveryInterval, logger)

	// the recovery proposal flow is for nodes to surface payloads that should
	// be recovered. these values are passed to the network and the network
	// votes on the proposed values
	svc1, recoveryProposer := newRecoveryProposalFlow(preprocessors, mStore, rp, recoveryInterval, logger, recoverConfigs...)

	// the retry flow is for payloads where the block number is still within
	// range of RPC data. this is a short range retry and failures here get
	// elevated to the recovery proposal flow.
	svc2, retryer := newRetryFlow(preprocessors, rStore, runner, recoveryProposer, recoveryInterval, logger, retryConfigs...)

	// the log trigger flow is the happy path for log trigger payloads. all
	// retryables that are encountered in this flow are elevated to the retry
	// flow
	svc3 := newLogTriggerFlow(preprocessors, rStore, runner, retryer, recoveryProposer, logProvider, logInterval, logger)

	// all above flows run internal time-keeper services. each is essential for
	// running so the return is a slice of all above services as recoverables
	svcs := []service.Recoverable{
		svc0,
		svc1,
		svc2,
		svc3,
	}

	// the final return includes a struct that provides the ability for hooks
	// to pass data to internal flows
	return &LogTriggerEligibility{
		builder:   builder,
		recoverer: recoverer,
		logger:    logger,
	}, svcs
}

// ProcessOutcome functions as an observation pre-build hook to allow data from
// outcomes to feed inputs in the eligibility flow
func (flow *LogTriggerEligibility) ProcessOutcome(outcome ocr2keepersv3.AutomationOutcome) error {
	var ok bool

	// if recoverable items are in outcome, proceed with values
	rawProposals, ok := outcome.Metadata[ocr2keepersv3.CoordinatedRecoveryProposalKey]
	if !ok {
		flow.logger.Printf("no proposed recoverables found in outcome")

		return nil
	}

	// proposals are trigger ids
	proposals, ok := rawProposals.([]ocr2keepers.CoordinatedProposal)
	if !ok {
		return fmt.Errorf("%w: coordinated proposals are not of type `string`", ErrWrongDataType)
	}

	// get latest coordinated block
	// by checking latest outcome first and then looping through the history
	var (
		rawBlock       interface{}
		blockAvailable bool
		block          ocr2keepers.BlockKey
	)

	if rawBlock, ok = outcome.Metadata[ocr2keepersv3.CoordinatedBlockOutcomeKey]; !ok {
		for _, h := range outcome.History {
			if rawBlock, ok = h.Metadata[ocr2keepersv3.CoordinatedBlockOutcomeKey]; !ok {
				continue
			}

			blockAvailable = true

			break
		}
	} else {
		blockAvailable = true
	}

	// we have proposals but a latest block isn't available
	if !blockAvailable {
		return ErrBlockNotAvailable
	}

	if block, ok = rawBlock.(ocr2keepers.BlockKey); !ok {
		return fmt.Errorf("%w: coordinated block value not of type `BlockKey`", ErrWrongDataType)
	}

	// limit timeout to get all proposal data
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

	// merge block number and recoverables
	for _, proposal := range proposals {
		proposal.Block = block

		payload, err := flow.builder.BuildPayload(ctx, proposal)
		if err != nil {
			flow.logger.Printf("error encountered when")
			continue
		}

		// pass to recoverer
		if err := flow.recoverer.Retry(ocr2keepers.CheckResult{
			Payload: payload,
		}); err != nil {
			continue
		}
	}

	cancel()

	return nil
}

type Scheduler[T any] interface {
	Schedule(string, T) error
}

type scheduledRetryer struct {
	scheduler Scheduler[ocr2keepers.UpkeepPayload]
}

func (s *scheduledRetryer) Retry(r ocr2keepers.CheckResult) error {
	if !r.Retryable {
		// exit condition for not retryable
		return fmt.Errorf("%w: %s", ErrNotRetryable, r.Payload.ID)
	}

	// TODO: validate that block is still valid for retry; if not error

	return s.scheduler.Schedule(r.Payload.ID, r.Payload)
}

type BasicRetryer[T any] interface {
	Add(string, T) error
}

type basicRetryer struct {
	ticker BasicRetryer[ocr2keepers.UpkeepPayload]
}

func (s *basicRetryer) Retry(r ocr2keepers.CheckResult) error {
	return s.ticker.Add(r.Payload.ID, r.Payload)
}

type logTick struct {
	logProvider LogEventProvider
	logger      *log.Logger
}

func (et logTick) Value(ctx context.Context) ([]ocr2keepers.UpkeepPayload, error) {
	if et.logProvider == nil {
		return nil, nil
	}

	logs, err := et.logProvider.GetLogs(ctx)

	et.logger.Printf("%d logs returned by log provider", len(logs))

	return logs, err
}

// log trigger flow is the happy path entry point for log triggered upkeeps
func newLogTriggerFlow(
	preprocessors []ocr2keepersv3.PreProcessor[ocr2keepers.UpkeepPayload],
	rs ResultStore,
	rn Runner,
	retryer Retryer,
	recoverer Retryer,
	logProvider LogEventProvider,
	logInterval time.Duration,
	logger *log.Logger,
) service.Recoverable {
	// postprocessing is a combination of multiple smaller postprocessors
	post := postprocessors.NewCombinedPostprocessor(
		// create eligibility postprocessor with result store
		postprocessors.NewEligiblePostProcessor(rs),
		// create retry postprocessor
		postprocessors.NewRetryPostProcessor(retryer, recoverer),
	)

	// create observer
	obs := ocr2keepersv3.NewRunnableObserver(preprocessors, post, rn, ObservationProcessLimit)

	// create time ticker
	timeTick := tickers.NewTimeTicker[[]ocr2keepers.UpkeepPayload](logInterval, obs, func(ctx context.Context, _ time.Time) (tickers.Tick[[]ocr2keepers.UpkeepPayload], error) {
		return logTick{logger: logger, logProvider: logProvider}, nil
	}, log.New(logger.Writer(), fmt.Sprintf("[%s | log-trigger-primary]", telemetry.ServiceName), telemetry.LogPkgStdFlags))

	return timeTick
}
