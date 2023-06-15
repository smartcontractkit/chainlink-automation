package ocr2keepers

import (
	"context"

	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/tickers"
)

// Preprocessor is the general interface for middleware used to filter, add, or modify upkeep
// payloads before checking their eligibility status
type Preprocessor interface {
	// PreProcess takes a slice of payloads and returns a new slice
	PreProcess(context.Context, []ocr2keepers.UpkeepPayload) ([]ocr2keepers.UpkeepPayload, error)
}

// Postprocessor is the general interface for a processing function after checking eligibility status
type Postprocessor interface {
	// PostProcess takes a slice of results where eligibility status is known
	PostProcess(context.Context, []ocr2keepers.CheckResult) error
}

// Runner is the interface for an object that should determine eligibility state
type Runner interface {
	// CheckUpkeeps has an input of upkeeps with unknown state and an output of upkeeps with known state
	CheckUpkeeps(context.Context, []ocr2keepers.UpkeepPayload) ([]ocr2keepers.CheckResult, error)
}

type Observer struct {
	Preprocessors []Preprocessor
	Postprocessor Postprocessor
	Runner        Runner
}

// NewObserver creates a new Observer with the given pre-processors, post-processor, and runner
func NewObserver(preprocessors []Preprocessor, postprocessor Postprocessor, runner Runner) *Observer {
	return &Observer{
		Preprocessors: preprocessors,
		Postprocessor: postprocessor,
		Runner:        runner,
	}
}

// Process - receives a tick and runs it through the eligibility pipeline. Calls all pre-processors, runs the check pipeline, and calls the post-processor.
func (o *Observer) Process(ctx context.Context, tick tickers.Tick) error {
	// Get upkeeps from tick
	upkeepPayloads, err := tick.GetUpkeeps(ctx)
	if err != nil {
		return err
	}

	// Run pre-processors
	for _, preprocessor := range o.Preprocessors {
		upkeepPayloads, err = preprocessor.PreProcess(ctx, upkeepPayloads)
		if err != nil {
			return err
		}
	}

	var results []ocr2keepers.CheckResult

	// Run check pipeline
	results, err = o.Runner.CheckUpkeeps(ctx, upkeepPayloads)
	if err != nil {
		return err
	}

	// Run post-processor
	return o.Postprocessor.PostProcess(ctx, results)
}

func (o *Observer) SetPostProcessor(pp Postprocessor) {
	o.Postprocessor = pp
}
