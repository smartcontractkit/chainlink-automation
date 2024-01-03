package ocr2keepers

import (
	"context"
	"log"
	"sync/atomic"
	"time"

	"github.com/smartcontractkit/chainlink-automation/pkg/v3/tickers"
	ocr2keepers "github.com/smartcontractkit/chainlink-automation/pkg/v3/types"
)

// PreProcessor is the general interface for middleware used to filter, add, or modify upkeep
// payloads before checking their eligibility status
type PreProcessor[T any] interface {
	// PreProcess takes a slice of payloads and returns a new slice
	PreProcess(context.Context, []T) ([]T, error)
}

// PostProcessor is the general interface for a processing function after checking eligibility status
type PostProcessor[T any] interface {
	// PostProcess takes a slice of results where eligibility status is known
	PostProcess(context.Context, []ocr2keepers.CheckResult, []T) error
}

// Runner is the interface for an object that should determine eligibility state
type Runner interface {
	// CheckUpkeeps has an input of upkeeps with unknown state and an output of upkeeps with known state
	CheckUpkeeps(context.Context, ...ocr2keepers.UpkeepPayload) ([]ocr2keepers.CheckResult, error)
}

type Observer[T any] struct {
	lggr *log.Logger

	Preprocessors []PreProcessor[T]
	Postprocessor PostProcessor[T]
	processFunc   func(context.Context, ...T) ([]ocr2keepers.CheckResult, error)

	// internal configurations
	processTimeLimit time.Duration

	gotFromTicker int64
	preProcessed  int64
	processed     int64
	postProcessed int64
}

// NewRunnableObserver creates a new Observer with the given pre-processors, post-processor, and runner
func NewRunnableObserver(
	preprocessors []PreProcessor[ocr2keepers.UpkeepPayload],
	postprocessor PostProcessor[ocr2keepers.UpkeepPayload],
	runner Runner,
	processLimit time.Duration,
	logger *log.Logger,
) *Observer[ocr2keepers.UpkeepPayload] {
	return &Observer[ocr2keepers.UpkeepPayload]{
		lggr:             logger,
		Preprocessors:    preprocessors,
		Postprocessor:    postprocessor,
		processFunc:      runner.CheckUpkeeps,
		processTimeLimit: processLimit,
	}
}

// NewGenericObserver creates a new Observer with the given pre-processors, post-processor, and runner
func NewGenericObserver[T any](
	preprocessors []PreProcessor[T],
	postprocessor PostProcessor[T],
	processor func(context.Context, ...T) ([]ocr2keepers.CheckResult, error),
	processLimit time.Duration,
	logger *log.Logger,
) *Observer[T] {
	return &Observer[T]{
		lggr:             logger,
		Preprocessors:    preprocessors,
		Postprocessor:    postprocessor,
		processFunc:      processor,
		processTimeLimit: processLimit,
	}
}

// Process - receives a tick and runs it through the eligibility pipeline. Calls all pre-processors, runs the check pipeline, and calls the post-processor.
func (o *Observer[T]) Process(ctx context.Context, tick tickers.Tick[[]T]) error {
	pCtx, cancel := context.WithTimeout(ctx, o.processTimeLimit)

	defer cancel()

	// Get upkeeps from tick
	value, err := tick.Value(pCtx)
	if err != nil {
		return err
	}

	gotFromTicker := atomic.AddInt64(&o.gotFromTicker, int64(len(value)))
	o.lggr.Printf("got %d payloads from ticker", len(value))

	// Run pre-processors
	for _, preprocessor := range o.Preprocessors {
		value, err = preprocessor.PreProcess(pCtx, value)
		if err != nil {
			return err
		}
	}

	preProcessed := atomic.AddInt64(&o.preProcessed, int64(len(value)))
	o.lggr.Printf("processing %d payloads", len(value))

	// Run check pipeline
	results, err := o.processFunc(pCtx, value...)
	if err != nil {
		return err
	}

	processed := atomic.AddInt64(&o.processed, int64(len(results)))
	o.lggr.Printf("post-processing %d results", len(results))

	// Run post-processor
	if err := o.Postprocessor.PostProcess(pCtx, results, value); err != nil {
		return err
	}

	postProcessed := atomic.AddInt64(&o.postProcessed, int64(len(results)))
	o.lggr.Printf("finished processing of %d results: %+v", len(results), results)

	o.lggr.Printf("logs stats: gotFromTicker %d; preProcessed %d; processed %d; postProcessed %d", gotFromTicker, preProcessed, processed, postProcessed)

	return nil
}
