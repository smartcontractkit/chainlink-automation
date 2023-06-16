package flows

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync/atomic"
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

// Recoverer provides the ability to push recoveries to an observer
type Recoverer interface {
	// Recover provides an entry point for new recoverable/retryable results
	Recover(ocr2keepers.CheckResult) error
}

const (
	LogCheckInterval      = 1 * time.Second
	RetryCheckInterval    = 250 * time.Millisecond
	RecoveryCheckInterval = 1 * time.Minute
)

// LogTriggerEligibility is a flow controller that surfaces eligible upkeeps
// with retry attempts.
type LogTriggerEligibility struct {
	// created in the constructor
	services []service.Recoverable

	// state variables
	running atomic.Bool
	chClose chan struct{}
}

// NewLogTriggerEligibility ...
func NewLogTriggerEligibility(logLookup PreProcessor, rStore ResultStore, runner Runner, logger *log.Logger, configFuncs ...tickers.RetryConfigFunc) *LogTriggerEligibility {
	svc0, recoverer := newRecoveryFlow(rStore, runner)
	svc1, retryer := newRetryFlow(rStore, runner, recoverer, configFuncs...)
	svc2 := newLogTriggerFlow(rStore, runner, retryer, recoverer, logLookup)

	return &LogTriggerEligibility{
		services: []service.Recoverable{
			service.NewRecoverer(svc0, logger),
			service.NewRecoverer(svc1, logger),
			service.NewRecoverer(svc2, logger),
		},
		chClose: make(chan struct{}, 1),
	}
}

// Start passes the provided context to dependent services and blocks until
// Close is called. If any errors are encountered in starting services, the
// errors will be joined and returned immediately.
func (flow *LogTriggerEligibility) Start(ctx context.Context) error {
	if flow.running.Load() {
		return fmt.Errorf("already running")
	}

	ctx, cancel := context.WithCancel(ctx)

	var err error

	// TODO: [AUTO-3414] what happens when service 1 starts successfully and
	// service 2 fails service 1 should be stopped and the error returned
	// immediately.
	for _, svc := range flow.services {
		err = errors.Join(err, svc.Start(ctx))
	}

	flow.running.Store(true)

	if err != nil {
		cancel()
		return err
	}

	select {
	case <-ctx.Done():
		cancel()
		return ctx.Err()
	case <-flow.chClose:
		cancel()
		return nil
	}
}

// Close ...
func (flow *LogTriggerEligibility) Close() error {
	if !flow.running.Load() {
		return fmt.Errorf("already stopped")
	}

	var err error

	for _, svc := range flow.services {
		err = errors.Join(err, svc.Close())
	}

	flow.running.Store(false)
	flow.chClose <- struct{}{}

	return err
}

// ProcessOutcome functions as an observation pre-build hook to allow data from
// outcomes to feed inputs in the eligibility flow
func (flow *LogTriggerEligibility) ProcessOutcome(_ ocr2keepersv3.AutomationOutcome) error {
	panic("log trigger observation pre-build hook not implemented")
}

func newRecoveryFlow(rs ResultStore, rn ocr2keepersv3.Runner, configFuncs ...tickers.RecoveryConfigFunc) (service.Recoverable, Recoverer) {
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

func newRetryFlow(rs ResultStore, rn ocr2keepersv3.Runner, rc Recoverer, configFuncs ...tickers.RetryConfigFunc) (service.Recoverable, Retryer) {
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
		postprocessors.NewRetryPostProcessor(ticker, rc),
	)

	retryObserver.SetPostProcessor(post)

	// return retry ticker and retryer (they are the same entity but satisfy two interfaces)
	return ticker, ticker
}

type emptyTick struct{}

func (et emptyTick) GetUpkeeps(context.Context) ([]ocr2keepers.UpkeepPayload, error) {
	return nil, nil
}

func newLogTriggerFlow(rs ResultStore, rn ocr2keepersv3.Runner, rt Retryer, rc Recoverer, logs PreProcessor) service.Recoverable {
	// postprocessing is a combination of multiple smaller postprocessors
	post := postprocessors.NewCombinedPostprocessor(
		// create eligibility postprocessor with result store
		postprocessors.NewEligiblePostProcessor(rs),
		// create retry postprocessor
		postprocessors.NewRetryPostProcessor(rt, rc),
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
