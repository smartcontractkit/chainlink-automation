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

const (
	LogCheckInterval        = 1 * time.Second
	RecoveryCheckInterval   = 1 * time.Minute
	ObservationProcessLimit = 5 * time.Second
)

// LogTriggerEligibility is a flow controller that surfaces eligible upkeeps
// with retry attempts.
type LogTriggerEligibility struct{}

// NewLogTriggerEligibility ...
func NewLogTriggerEligibility(
	coord PreProcessor,
	rStore ResultStore,
	mStore MetadataStore,
	runner Runner,
	logProvider LogEventProvider,
	rp RecoverableProvider,
	logInterval time.Duration,
	recoveryInterval time.Duration,
	logger *log.Logger,
	retryConfigs []tickers.ScheduleTickerConfigFunc,
	recoverConfigs []tickers.ScheduleTickerConfigFunc,
) (*LogTriggerEligibility, []service.Recoverable) {
	preprocessors := []ocr2keepersv3.PreProcessor[ocr2keepers.UpkeepPayload]{coord}
	svc0, recoverer := newRecoveryProposalFlow(preprocessors, mStore, rp, recoveryInterval, logger, recoverConfigs...)
	svc1, retryer := newRetryFlow(preprocessors, rStore, runner, recoverer, recoveryInterval, logger, retryConfigs...)
	svc2 := newLogTriggerFlow(preprocessors, rStore, runner, retryer, recoverer, logProvider, logInterval, logger)

	svcs := []service.Recoverable{
		svc0,
		svc1,
		svc2,
	}

	return &LogTriggerEligibility{}, svcs
}

// ProcessOutcome functions as an observation pre-build hook to allow data from
// outcomes to feed inputs in the eligibility flow
func (flow *LogTriggerEligibility) ProcessOutcome(_ ocr2keepersv3.AutomationOutcome) error {
	// panic("log trigger observation pre-build hook not implemented")

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
