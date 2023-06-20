package flows

import (
	"context"
	"log"
	"time"

	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg"
	ocr2keepersv3 "github.com/smartcontractkit/ocr2keepers/pkg/v3"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/postprocessors"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/service"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/tickers"
)

//go:generate mockery --name Runner --structname MockRunner --srcpkg "github.com/smartcontractkit/ocr2keepers/pkg/v3/flows" --case underscore --filename runner.generated.go
type Runner interface {
	// CheckUpkeeps has an input of upkeeps with unknown state and an output of upkeeps with known state
	CheckUpkeeps(context.Context, []ocr2keepers.UpkeepPayload) ([]ocr2keepers.CheckResult, error)
}

//go:generate mockery --name PreProcessor --structname MockPreProcessor --srcpkg "github.com/smartcontractkit/ocr2keepers/pkg/v3/flows" --case underscore --filename preprocessor.generated.go
type PreProcessor interface {
	// PreProcess takes a slice of payloads and returns a new slice
	PreProcess(context.Context, []ocr2keepers.UpkeepPayload) ([]ocr2keepers.UpkeepPayload, error)
}

//go:generate mockery --name ResultStore --structname MockResultStore --srcpkg "github.com/smartcontractkit/ocr2keepers/pkg/v3/flows" --case underscore --filename resultstore.generated.go
type ResultStore interface {
	Add(...ocr2keepers.CheckResult)
}

// Retryer provides the ability to push retries to an observer
type Retryer interface {
	// Retry provides an entry point for new retryable results
	Retry(ocr2keepers.CheckResult) error
}

const (
	LogCheckInterval      = 1 * time.Second
	RetryCheckInterval    = 250 * time.Millisecond
	RecoveryCheckInterval = 1 * time.Minute
)

// LogTriggerEligibility is a flow controller that surfaces eligible upkeeps
// with retry attempts.
type LogTriggerEligibility struct{}

// NewLogTriggerEligibility ...
func NewLogTriggerEligibility(logLookup PreProcessor, rStore ResultStore, runner Runner, _ *log.Logger, configFuncs ...tickers.RetryConfigFunc) (*LogTriggerEligibility, []service.Recoverable) {
	svc0, recoverer := newRecoveryFlow(rStore, runner)
	svc1, retryer := newRetryFlow(rStore, runner, recoverer, configFuncs...)
	svc2 := newLogTriggerFlow(rStore, runner, retryer, recoverer, logLookup)

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
	panic("log trigger observation pre-build hook not implemented")
}

func newRecoveryFlow(rs ResultStore, rn ocr2keepersv3.Runner, configFuncs ...tickers.RetryConfigFunc) (service.Recoverable, Retryer) {
	// create observer
	// no preprocessors required for retry flow at this point
	// leave postprocessor empty to start with
	recoveryObserver := ocr2keepersv3.NewObserver(nil, nil, rn)

	// create retry ticker
	ticker := tickers.NewRecoveryTicker(RecoveryCheckInterval, recoveryObserver, configFuncs...)

	// postprocess
	post := postprocessors.NewEligiblePostProcessor(rs)

	recoveryObserver.SetPostProcessor(post)

	// return retry ticker and retryer (they are the same entity but satisfy two interfaces)
	return ticker, ticker
}

func newRetryFlow(rs ResultStore, rn ocr2keepersv3.Runner, recoverer Retryer, configFuncs ...tickers.RetryConfigFunc) (service.Recoverable, Retryer) {
	// create observer
	// no preprocessors required for retry flow at this point
	// leave postprocessor empty to start with
	retryObserver := ocr2keepersv3.NewObserver(nil, nil, rn)

	// create retry ticker
	ticker := tickers.NewRetryTicker(RetryCheckInterval, retryObserver, configFuncs...)

	// postprocessing is a combination of multiple smaller postprocessors
	post := postprocessors.NewCombinedPostprocessor(
		// create eligibility postprocessor with result store
		postprocessors.NewEligiblePostProcessor(rs),
		// create retry postprocessor
		postprocessors.NewRetryPostProcessor(ticker, recoverer),
	)

	retryObserver.SetPostProcessor(post)

	// return retry ticker and retryer (they are the same entity but satisfy two interfaces)
	return ticker, ticker
}

type emptyTick struct{}

func (et emptyTick) GetUpkeeps(context.Context) ([]ocr2keepers.UpkeepPayload, error) {
	return nil, nil
}

func newLogTriggerFlow(rs ResultStore, rn ocr2keepersv3.Runner, retryer Retryer, recoverer Retryer, logs PreProcessor) service.Recoverable {
	// postprocessing is a combination of multiple smaller postprocessors
	post := postprocessors.NewCombinedPostprocessor(
		// create eligibility postprocessor with result store
		postprocessors.NewEligiblePostProcessor(rs),
		// create retry postprocessor
		postprocessors.NewRetryPostProcessor(retryer, recoverer),
	)

	// create observer
	obs := ocr2keepersv3.NewObserver([]ocr2keepersv3.PreProcessor{logs}, post, rn)

	// create time ticker
	timeTick := tickers.NewTimeTicker(LogCheckInterval, obs, func(context.Context, time.Time) (tickers.Tick, error) {
		// getter function returns empty set to allow first postprocessor
		// to query the registry
		return emptyTick{}, nil
	})

	return timeTick
}
