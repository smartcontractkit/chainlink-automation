package ocrtypes

import (
	"context"

	"github.com/smartcontractkit/libocr/offchainreporting2plus/types"
)

type InstructionStore interface{}

// SamplingStore at this point can be an empty interface
type SamplingStore interface{}

// ResultStore at this point can be an empty interface
type ResultStore interface{}

type ocr3Plugin struct {
	PrebuildHooks     []func(AutomationOutcome) error
	BuildHooks        []func(*AutomationObservation, InstructionStore, SamplingStore, ResultStore) error
	InstructionSource InstructionStore
	MetadataSource    SamplingStore
	ResultSource      ResultStore
}

// OutcomeContext is imported from libocr
func (plugin *ocr3Plugin) Observation(ctx context.Context, outcome OutcomeContext, query types.Query) (types.Observation, error) {
	// Decode the outcome to AutomationOutcome
	var automationOutcome AutomationOutcome
	// this function is part of AutomationOutcome struct
	err := DecodeAutomationOutcome(outcome.Bytes, &automationOutcome)
	if err != nil {
		return nil, err
	}

	// Execute pre-build hooks
	for _, hook := range plugin.PrebuildHooks {
		err := hook(automationOutcome)
		if err != nil {
			return nil, err
		}
	}

	// Create new AutomationObservation
	observation := AutomationObservation{
		Instructions: automationOutcome.Instructions,
		Metadata:     automationOutcome.Metadata,
		Performable:  automationOutcome.Performable,
	}

	// Execute build hooks
	for _, hook := range plugin.BuildHooks {
		err := hook(&observation, plugin.InstructionSource, plugin.MetadataSource, plugin.ResultSource)
		if err != nil {
			return nil, err
		}
	}

	// Encode the observation to bytes
	encoded, err := observation.EncodeAutomationObservation()
	if err != nil {
		return nil, err
	}

	// Return the encoded bytes as ocr3 observation
	return types.NewObservation(encoded), nil
}
