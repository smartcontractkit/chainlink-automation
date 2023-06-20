package plugin

import (
	"context"
	"testing"

	"github.com/smartcontractkit/libocr/offchainreporting2plus/ocr3types"
	"github.com/smartcontractkit/libocr/offchainreporting2plus/types"
	"github.com/stretchr/testify/assert"

	ocr2keepersv3 "github.com/smartcontractkit/ocr2keepers/pkg/v3"
)

func TestObservation(t *testing.T) {
	// Create an instance of ocr3 plugin
	plugin := &ocr3Plugin[int]{}

	// Create a sample outcome for decoding
	outcome := ocr3types.OutcomeContext{
		PreviousOutcome: []byte(`{"Instructions":["instruction1"],"Metadata":{"key":"value"},"Performable":[]}`),
	}

	// Define a mock hook function for testing pre-build hooks
	mockPrebuildHook := func(outcome ocr2keepersv3.AutomationOutcome) error {
		assert.Equal(t, 1, len(outcome.Instructions))
		return nil
	}

	// Add the mock pre-build hook to the plugin's PrebuildHooks
	plugin.PrebuildHooks = append(plugin.PrebuildHooks, mockPrebuildHook)

	// Define a mock build hook function for testing build hooks
	mockBuildHook := func(observation *ocr2keepersv3.AutomationObservation, instructionStore ocr2keepersv3.InstructionStore, samplingStore ocr2keepersv3.SamplingStore, resultStore ocr2keepersv3.ResultStore) error {
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
