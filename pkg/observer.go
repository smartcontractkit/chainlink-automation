package ocr2keepers

import "context"

type UpkeepPayload interface{}
type CheckResult interface{}

type Tick interface {
	// GetUpkeeps provides an array of upkeeps scoped to the individual tick
	GetUpkeeps(ctx context.Context) ([]UpkeepPayload, error)
}

// Runner2 is the interface for an object that should determine eligibility state
// (I didn't want to mess with the existing runner and was unsure if we wanted to reuse that)
type Runner2 interface {
	// CheckUpkeeps has an input of upkeeps with unknown state and an output of upkeeps with known state
	CheckUpkeeps(context.Context, []UpkeepPayload) ([]CheckResult, error)
}

// Preprocessor is the general interface for middleware used to filter, add, or modify upkeep
// payloads before checking their eligibility status
type Preprocessor interface {
	// PreProcess takes a slice of payloads and returns a new slice
	PreProcess(context.Context, []UpkeepPayload) ([]UpkeepPayload, error)
}

// Postprocessor is the general interface for a processing function after checking eligibility status
type Postprocessor interface {
	// PostProcess takes a slice of results where eligibility status is known
	PostProcess(context.Context, []CheckResult) error
}

type Observer interface {
	// Process should use an arbitrary tick as input
	Process(context.Context, Tick) error
}

type Observe struct {
	Preprocessors []Preprocessor
	Postprocessor Postprocessor
	Runner        Runner2
}

// NewObserver creates a new Observer with the given pre-processors, post-processor, and runner
func NewObserver(preprocessors []Preprocessor, postprocessor Postprocessor, runner Runner2) Observer {
	return &Observe{
		Preprocessors: preprocessors,
		Postprocessor: postprocessor,
		Runner:        runner,
	}
}

// Process - receives a tick and runs it through the eligibility pipeline. Calls all pre-processors, runs the check pipeline, and calls the post-processor.
func (o *Observe) Process(ctx context.Context, tick Tick) error {
	// Get upkeeps from tick
	upkeeps, err := tick.GetUpkeeps(ctx)
	if err != nil {
		return err
	}

	var upkeepPayloads []UpkeepPayload
	// Run pre-processors
	for _, preprocessor := range o.Preprocessors {
		upkeepPayloads, err = preprocessor.PreProcess(ctx, upkeeps)
		if err != nil {
			return err
		}
	}

	// Run check pipeline
	results, err := o.Runner.CheckUpkeeps(ctx, upkeepPayloads)
	if err != nil {
		return err
	}

	// Run post-processor
	err = o.Postprocessor.PostProcess(ctx, results)
	if err != nil {
		return err
	}

	return nil
}
