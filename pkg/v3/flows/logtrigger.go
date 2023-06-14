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
	Remove(...ocr2keepers.CheckResult)
	View() ([]ocr2keepers.CheckResult, error)
}

// Retryer provides the ability to push retries to an observer
type Retryer interface {
	// Retry provides an entry point for new retryable results
	Retry(ocr2keepers.CheckResult) error
}

const (
	LogCheckInterval   = 1 * time.Second
	RetryCheckInterval = 250 * time.Millisecond
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
func NewLogTriggerEligibility(logLookup PreProcessor, rStore ResultStore, runner Runner, logger *log.Logger) *LogTriggerEligibility {
	svc, retryer := newRetryFlow(rStore, runner)
	svc2 := newLogTriggerFlow(rStore, runner, retryer, logLookup)

	return &LogTriggerEligibility{
		services: []service.Recoverable{
			service.NewRecoverer(svc, logger),
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

	var err error

	// TODO: [AUTO-3414] what happens when service 1 starts successfully and
	// service 2 fails service 1 should be stopped and the error returned
	// immediately.
	for _, svc := range flow.services {
		err = errors.Join(err, svc.Start(ctx))
	}

	flow.running.Store(true)

	if err != nil {
		return err
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-flow.chClose:
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

func newRetryFlow(rs ResultStore, rn ocr2keepersv3.Runner) (service.Recoverable, Retryer) {
	// create observer
	// no preprocessors required for retry flow at this point
	// leave postprocessor empty to start with
	retryObserver := ocr2keepersv3.NewObserver(nil, nil, rn)

	// create retry ticker
	ticker := tickers.NewRetryTicker(RetryCheckInterval, retryObserver)

	// postprocessing is a combination of multiple smaller postprocessors
	post := postprocessors.NewCombinedPostprocessor(
		// create eligibility postprocessor with result store
		postprocessors.NewEligiblePostProcessor(rs),
		// create retry postprocessor
		postprocessors.NewRetryPostProcessor(ticker),
	)

	retryObserver.SetPostProcessor(post)

	// return retry ticker and retryer (they are the same entity but satisfy two interfaces)
	return ticker, ticker
}

type emptyTick struct{}

func (et emptyTick) GetUpkeeps(context.Context) ([]ocr2keepers.UpkeepPayload, error) {
	return nil, nil
}

func newLogTriggerFlow(rs ResultStore, rn ocr2keepersv3.Runner, rt Retryer, logs PreProcessor) service.Recoverable {
	// postprocessing is a combination of multiple smaller postprocessors
	post := postprocessors.NewCombinedPostprocessor(
		// create eligibility postprocessor with result store
		postprocessors.NewEligiblePostProcessor(rs),
		// create retry postprocessor
		postprocessors.NewRetryPostProcessor(rt),
	)

	// create observer
	obs := ocr2keepersv3.NewObserver([]ocr2keepersv3.Preprocessor{logs}, post, rn)

	// create time ticker
	timeTick := tickers.NewTimeTicker(LogCheckInterval, obs, func(context.Context, time.Time) (tickers.Tick, error) {
		// getter function returns empty set to allow first postprocessor
		// to query the registry
		return emptyTick{}, nil
	})

	return timeTick
}
