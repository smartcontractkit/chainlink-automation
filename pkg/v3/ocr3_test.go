package v3

import (
	"context"
	"testing"

	"github.com/smartcontractkit/libocr/offchainreporting2plus/ocr3types"
	"github.com/stretchr/testify/assert"

	"github.com/smartcontractkit/libocr/offchainreporting2plus/types"
)

func TestObservation(t *testing.T) {
	// Create an instance of ocr3 plugin
	plugin := &ocr3Plugin{}

	// Create a sample outcome for decoding
	outcome := ocr3types.OutcomeContext{
		PreviousOutcome: []byte(`{"Instructions":["instruction1"],"Metadata":{"key":"value"},"Performable":[]}`),
	}

	// Define a mock hook function for testing pre-build hooks
	mockPrebuildHook := func(outcome AutomationOutcome) error {
		assert.Equal(t, 1, len(outcome.Instructions))
		return nil
	}

	// Add the mock pre-build hook to the plugin's PrebuildHooks
	plugin.PrebuildHooks = append(plugin.PrebuildHooks, mockPrebuildHook)

	// Define a mock build hook function for testing build hooks
	mockBuildHook := func(observation *AutomationObservation, instructionStore InstructionStore, samplingStore SamplingStore, resultStore ResultStore) error {
		assert.Equal(t, 0, len(observation.Instructions))
		return nil
	}

	// Add the mock build hook to the plugin's BuildHooks
	plugin.BuildHooks = append(plugin.BuildHooks, mockBuildHook)

	// Create a sample query for testing
	query := types.Query{}

	// Call the Observation function
	observation, err := plugin.Observation(context.Background(), outcome, query)
	assert.NoError(t, err)

	// Assert that the returned observation matches the expected encoded outcome
	expectedEncodedOutcome := []byte(`{"Instructions":null,"Metadata":null,"Performable":null}`)
	assert.Equal(t, types.Observation(expectedEncodedOutcome), observation)
}
