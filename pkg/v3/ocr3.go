package v3

import (
	"context"
	"errors"

	"github.com/smartcontractkit/libocr/offchainreporting2plus/ocr3types"
	"github.com/smartcontractkit/libocr/offchainreporting2plus/types"
)

type InstructionStore interface{}

type SamplingStore interface{}

type ResultStore interface{}

type ocr3Plugin struct {
	PrebuildHooks     []func(AutomationOutcome) error
	BuildHooks        []func(*AutomationObservation, InstructionStore, SamplingStore, ResultStore) error
	InstructionSource InstructionStore
	MetadataSource    SamplingStore
	ResultSource      ResultStore
}

func (plugin *ocr3Plugin) Observation(ctx context.Context, outcome ocr3types.OutcomeContext, query types.Query) (types.Observation, error) {
	// Decode the outcome to AutomationOutcome
	automationOutcome, err := DecodeAutomationOutcome(outcome.PreviousOutcome)
	if err != nil {
		return nil, err
	}

	// Execute pre-build hooks
	for _, hook := range plugin.PrebuildHooks {
		err = errors.Join(err, hook(automationOutcome))
	}
	if err != nil {
		return nil, err
	}

	// Create new AutomationObservation
	observation := AutomationObservation{}

	// Execute build hooks
	for _, hook := range plugin.BuildHooks {
		err := hook(&observation, plugin.InstructionSource, plugin.MetadataSource, plugin.ResultSource)
		if err != nil {
			return nil, err
		}
	}

	// Encode the observation to bytes
	encoded, err := observation.Encode()
	if err != nil {
		return nil, err
	}

	// Return the encoded bytes as ocr3 observation
	return types.Observation(encoded), nil
}
