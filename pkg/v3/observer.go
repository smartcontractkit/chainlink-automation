package ocr2keepers

import (
	"context"
	"log"
	"time"

	ocr2keepers "github.com/smartcontractkit/chainlink-common/pkg/types/automation"

	"github.com/smartcontractkit/chainlink-automation/pkg/v3/tickers"
)

// PreProcessor is the general interface for middleware used to filter, add, or modify upkeep
// payloads before checking their eligibility status
type PreProcessor interface {
	// PreProcess takes a slice of payloads and returns a new slice
	PreProcess(context.Context, []ocr2keepers.UpkeepPayload) ([]ocr2keepers.UpkeepPayload, error)
}

// PostProcessor is the general interface for a processing function after checking eligibility status
type PostProcessor interface {
	// PostProcess takes a slice of results where eligibility status is known
	PostProcess(context.Context, []ocr2keepers.CheckResult, []ocr2keepers.UpkeepPayload) error
}

// Runner is the interface for an object that should determine eligibility state
type Runner interface {
	// CheckUpkeeps has an input of upkeeps with unknown state and an output of upkeeps with known state
	CheckUpkeeps(context.Context, ...ocr2keepers.UpkeepPayload) ([]ocr2keepers.CheckResult, error)
}

type Observer struct {
	lggr *log.Logger

	Preprocessors []PreProcessor
	Postprocessor PostProcessor
	processFunc   func(context.Context, ...ocr2keepers.UpkeepPayload) ([]ocr2keepers.CheckResult, error)

	// internal configurations
	processTimeLimit time.Duration
}

type SampleProposalObserverV2 struct {
	lggr *log.Logger

	Preprocessors []PreProcessor
	Postprocessor PostProcessor
	processFunc   func(context.Context, ...ocr2keepers.UpkeepPayload) ([]ocr2keepers.CheckResult, error)

	// internal configurations
	processTimeLimit time.Duration
}

// NewRunnableObserver creates a new Observer with the given pre-processors, post-processor, and runner
func NewRunnableObserver(
	preprocessors []PreProcessor,
	postprocessor PostProcessor,
	runner Runner,
	processLimit time.Duration,
	logger *log.Logger,
) *Observer {
	return &Observer{
		lggr:             logger,
		Preprocessors:    preprocessors,
		Postprocessor:    postprocessor,
		processFunc:      runner.CheckUpkeeps,
		processTimeLimit: processLimit,
	}
}

// NewRunnableObserver creates a new Observer with the given pre-processors, post-processor, and runner
func NewSampleProposalObserverV2(
	preprocessors []PreProcessor,
	postprocessor PostProcessor,
	runner Runner,
	processLimit time.Duration,
	logger *log.Logger,
) *SampleProposalObserverV2 {
	return &SampleProposalObserverV2{
		lggr:             logger,
		Preprocessors:    preprocessors,
		Postprocessor:    postprocessor,
		processFunc:      runner.CheckUpkeeps,
		processTimeLimit: processLimit,
	}
}

func (t *SampleProposalObserverV2) Start(pctx context.Context) error {
	return nil
}

func (t *SampleProposalObserverV2) Close() error {
	return nil
}

// Process - receives a tick and runs it through the eligibility pipeline. Calls all pre-processors, runs the check pipeline, and calls the post-processor.
func (o *Observer) Process(ctx context.Context, tick tickers.Tick) error {
	pCtx, cancel := context.WithTimeout(ctx, o.processTimeLimit)

	defer cancel()

	// Get upkeeps from tick
	value, err := tick.Value(pCtx)
	if err != nil {
		return err
	}

	o.lggr.Printf("got %d payloads from ticker", len(value))

	// Run pre-processors
	for _, preprocessor := range o.Preprocessors {
		value, err = preprocessor.PreProcess(pCtx, value)
		if err != nil {
			return err
		}
	}

	o.lggr.Printf("processing %d payloads", len(value))

	// Run check pipeline
	results, err := o.processFunc(pCtx, value...)
	if err != nil {
		return err
	}

	o.lggr.Printf("post-processing %d results", len(results))

	// Run post-processor
	if err := o.Postprocessor.PostProcess(pCtx, results, value); err != nil {
		return err
	}

	o.lggr.Printf("finished processing of %d results: %+v", len(results), results)

	return nil
}
