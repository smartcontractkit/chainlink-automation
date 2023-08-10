package plugin

import (
	"context"
	"io"
	"log"
	"testing"

	"github.com/smartcontractkit/libocr/offchainreporting2plus/ocr3types"
	"github.com/smartcontractkit/libocr/offchainreporting2plus/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	ocr2keepersv3 "github.com/smartcontractkit/ocr2keepers/pkg/v3"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/config"
	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg/v3/types"
	mocks2 "github.com/smartcontractkit/ocr2keepers/pkg/v3/types/mocks"
)

func TestObservation(t *testing.T) {
	// Create an instance of ocr3 plugin
	plugin := &ocr3Plugin{
		Logger: log.New(io.Discard, "", 0),
	}

	// Create a sample outcome for decoding
	outcome := ocr3types.OutcomeContext{
		PreviousOutcome: []byte(`{"Instructions":["do coordinate block"],"Metadata":{"blockHistory":["4"]},"Performable":[]}`),
	}

	// Define a mock hook function for testing pre-build hooks
	mockPrebuildHook := func(outcome ocr2keepersv3.AutomationOutcome) error {
		assert.Equal(t, 1, len(outcome.Instructions))
		return nil
	}

	// Add the mock pre-build hook to the plugin's PrebuildHooks
	plugin.PrebuildHooks = append(plugin.PrebuildHooks, mockPrebuildHook)

	// Define a mock build hook function for testing build hooks
	mockBuildHook := func(observation *ocr2keepersv3.AutomationObservation) error {
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
	expectedEncodedOutcome := []byte(`{"Instructions":null,"Metadata":{},"Performable":null}`)
	assert.Equal(t, types.Observation(expectedEncodedOutcome), observation)
}

func TestOcr3Plugin_Outcome(t *testing.T) {
	t.Run("malformed observations returns an error", func(t *testing.T) {
		// Create an instance of ocr3 plugin
		plugin := &ocr3Plugin{
			Logger: log.New(io.Discard, "", 0),
		}

		// Create a sample outcome for decoding
		outcomeContext := ocr3types.OutcomeContext{
			PreviousOutcome: []byte(`{"Instructions":["instruction1"],"Metadata":{"key":"value"},"Performable":[]}`),
		}

		observations := []types.AttributedObservation{
			{
				Observation: []byte("malformed"),
			},
		}

		outcome, err := plugin.Outcome(outcomeContext, nil, observations)
		assert.Nil(t, outcome)
		assert.Error(t, err)
	})

	// TODO disable this for now as outcomes will be reworked
	//t.Run("given three observations, in which two are identical, one observations is added to the outcome", func(t *testing.T) {
	//	// Create an instance of ocr3 plugin
	//	plugin := &ocr3Plugin{
	//		Logger: log.New(io.Discard, "", 0),
	//	}
	//
	//	// Create a sample outcome for decoding
	//	outcomeContext := ocr3types.OutcomeContext{
	//		PreviousOutcome: []byte(`{"Instructions":["should coordinate block"],"Metadata":{"blockHistory":[]},"Performable":[]}`),
	//	}
	//
	//	automationObservation1 := ocr2keepersv3.AutomationObservation{
	//		Performable: []ocr2keepers.CheckResult{
	//			{
	//				Eligible:     true,
	//				Retryable:    false,
	//				GasAllocated: 10,
	//				UpkeepID:     ocr2keepers.UpkeepIdentifier([32]byte{4}),
	//				Trigger: ocr2keepers.Trigger{
	//					BlockNumber: 4,
	//					BlockHash:   [32]byte{0},
	//					LogTriggerExtension: &ocr2keepers.LogTriggerExtenstion{
	//						LogTxHash: [32]byte{1},
	//						Index:     4,
	//					},
	//				},
	//			},
	//		},
	//	}
	//	automationObservation2 := ocr2keepersv3.AutomationObservation{
	//		Performable: []ocr2keepers.CheckResult{
	//			{
	//				Eligible:     true,
	//				Retryable:    false,
	//				GasAllocated: 10,
	//				UpkeepID:     ocr2keepers.UpkeepIdentifier([32]byte{4}),
	//				Trigger: ocr2keepers.Trigger{
	//					BlockNumber: 4,
	//					BlockHash:   [32]byte{0},
	//					LogTriggerExtension: &ocr2keepers.LogTriggerExtenstion{
	//						LogTxHash: [32]byte{1},
	//						Index:     4,
	//					},
	//				},
	//			},
	//		},
	//	}
	//	automationObservation3 := ocr2keepersv3.AutomationObservation{
	//		Performable: []ocr2keepers.CheckResult{
	//			{
	//				Eligible:     true,
	//				Retryable:    false,
	//				GasAllocated: 10,
	//				UpkeepID:     ocr2keepers.UpkeepIdentifier([32]byte{4}),
	//				Trigger: ocr2keepers.Trigger{
	//					BlockNumber: 4,
	//					BlockHash:   [32]byte{0},
	//					LogTriggerExtension: &ocr2keepers.LogTriggerExtenstion{
	//						LogTxHash: [32]byte{1},
	//						Index:     4,
	//					},
	//				},
	//			},
	//		},
	//	}
	//
	//	obs1, err := automationObservation1.Encode()
	//	assert.NoError(t, err)
	//	obs2, err := automationObservation2.Encode()
	//	assert.NoError(t, err)
	//	obs3, err := automationObservation3.Encode()
	//	assert.NoError(t, err)
	//
	//	observations := []types.AttributedObservation{
	//		{
	//			Observation: obs1,
	//		},
	//		{
	//			Observation: obs2, // payload matches obs1, giving 2 counts of the same payload
	//		},
	//		{
	//			Observation: obs3, // this single report instance won't reach the quorum threshold
	//		},
	//	}
	//
	//	outcome, err := plugin.Outcome(outcomeContext, nil, observations)
	//	assert.NoError(t, err)
	//
	//	automationOutcome, err := ocr2keepersv3.DecodeAutomationOutcome(outcome)
	//	assert.NoError(t, err)
	//
	//	// obs1 and obs2 are identical, so they will be considered the same result. obs3 doesn't reach the quorum threshold
	//	assert.Len(t, automationOutcome.Performable, 1)
	//})
}

func TestReports(t *testing.T) {
	t.Run("1 report less than limit; 1 report per batch", func(t *testing.T) {
		me := new(mocks2.MockEncoder)

		// Create an instance of ocr3 plugin
		plugin := &ocr3Plugin{
			Logger:        log.New(io.Discard, "", 0),
			ReportEncoder: me,
			Config: config.OffchainConfig{
				GasOverheadPerUpkeep: 100_000,
				GasLimitPerReport:    5_000_000,
				MaxUpkeepBatchSize:   1,
			},
		}

		outcome := ocr2keepersv3.AutomationOutcome{
			BasicOutcome: ocr2keepersv3.BasicOutcome{
				Performable: []ocr2keepers.CheckResult{
					{
						UpkeepID: ocr2keepers.UpkeepIdentifier([32]byte{1}),
						Trigger: ocr2keepers.Trigger{
							BlockNumber: 4,
							BlockHash:   [32]byte{0},
							LogTriggerExtension: &ocr2keepers.LogTriggerExtension{
								TxHash: [32]byte{1},
								Index:  4,
							},
						},
						GasAllocated: 1_000_000,
						PerformData:  []byte(`0x`),
					},
				},
			},
		}

		me.On("Encode", mock.Anything).Return([]byte(``), nil)

		rawOutcome, err := outcome.Encode()
		assert.NoError(t, err, "no error during encoding")

		reports, err := plugin.Reports(2, rawOutcome)
		assert.NoError(t, err, "no error from generating reports")
		assert.Len(t, reports, 1, "report length should be 1")

		me.AssertExpectations(t)
	})

	t.Run("2 reports less than limit; 1 report per batch", func(t *testing.T) {
		me := new(mocks2.MockEncoder)

		// Create an instance of ocr3 plugin
		plugin := &ocr3Plugin{
			Logger:        log.New(io.Discard, "", 0),
			ReportEncoder: me,
			Config: config.OffchainConfig{
				GasOverheadPerUpkeep: 100_000,
				GasLimitPerReport:    5_000_000,
				MaxUpkeepBatchSize:   1,
			},
		}

		outcome := ocr2keepersv3.AutomationOutcome{
			BasicOutcome: ocr2keepersv3.BasicOutcome{
				Performable: []ocr2keepers.CheckResult{
					{
						UpkeepID: ocr2keepers.UpkeepIdentifier([32]byte{1}),
						Trigger: ocr2keepers.Trigger{
							BlockNumber: 4,
							BlockHash:   [32]byte{0},
							LogTriggerExtension: &ocr2keepers.LogTriggerExtension{
								TxHash: [32]byte{1},
								Index:  4,
							},
						},
						GasAllocated: 1_000_000,
						PerformData:  []byte(`0x`),
					},
					{
						UpkeepID: ocr2keepers.UpkeepIdentifier([32]byte{1}),
						Trigger: ocr2keepers.Trigger{
							BlockNumber: 4,
							BlockHash:   [32]byte{0},
							LogTriggerExtension: &ocr2keepers.LogTriggerExtension{
								TxHash: [32]byte{1},
								Index:  4,
							},
						},
						GasAllocated: 1_000_000,
						PerformData:  []byte(`0x`),
					},
				},
			},
		}

		me.On("Encode", mock.Anything).Return([]byte(``), nil).Times(2)

		rawOutcome, err := outcome.Encode()
		assert.NoError(t, err, "no error during encoding")

		reports, err := plugin.Reports(2, rawOutcome)
		assert.NoError(t, err, "no error from generating reports")
		assert.Len(t, reports, 2, "report length should be 2")

		me.AssertExpectations(t)
	})

	t.Run("3 reports less than limit; 2 report per batch", func(t *testing.T) {
		me := new(mocks2.MockEncoder)

		// Create an instance of ocr3 plugin
		plugin := &ocr3Plugin{
			Logger:        log.New(io.Discard, "", 0),
			ReportEncoder: me,
			Config: config.OffchainConfig{
				GasOverheadPerUpkeep: 100_000,
				GasLimitPerReport:    5_000_000,
				MaxUpkeepBatchSize:   2,
			},
		}

		outcome := ocr2keepersv3.AutomationOutcome{
			BasicOutcome: ocr2keepersv3.BasicOutcome{
				Performable: []ocr2keepers.CheckResult{
					{
						UpkeepID: ocr2keepers.UpkeepIdentifier([32]byte{1}),
						Trigger: ocr2keepers.Trigger{
							BlockNumber: 4,
							BlockHash:   [32]byte{0},
							LogTriggerExtension: &ocr2keepers.LogTriggerExtension{
								TxHash: [32]byte{1},
								Index:  4,
							},
						},
						GasAllocated: 1_000_000,
						PerformData:  []byte(`0x`),
					},
					{
						UpkeepID: ocr2keepers.UpkeepIdentifier([32]byte{1}),
						Trigger: ocr2keepers.Trigger{
							BlockNumber: 4,
							BlockHash:   [32]byte{0},
							LogTriggerExtension: &ocr2keepers.LogTriggerExtension{
								TxHash: [32]byte{1},
								Index:  4,
							},
						},
						GasAllocated: 1_000_000,
						PerformData:  []byte(`0x`),
					},
					{
						UpkeepID: ocr2keepers.UpkeepIdentifier([32]byte{1}),
						Trigger: ocr2keepers.Trigger{
							BlockNumber: 4,
							BlockHash:   [32]byte{0},
							LogTriggerExtension: &ocr2keepers.LogTriggerExtension{
								TxHash: [32]byte{1},
								Index:  4,
							},
						},
						GasAllocated: 1_000_000,
						PerformData:  []byte(`0x`),
					},
				},
			},
		}

		me.On("Encode", mock.Anything, mock.Anything).Return([]byte(``), nil).Times(1)
		me.On("Encode", mock.Anything).Return([]byte(``), nil).Times(1)

		rawOutcome, err := outcome.Encode()
		assert.NoError(t, err, "no error during encoding")

		reports, err := plugin.Reports(2, rawOutcome)
		assert.NoError(t, err, "no error from generating reports")
		assert.Len(t, reports, 2, "report length should be 2")

		me.AssertExpectations(t)
	})

	t.Run("gas allocated larger than report limit", func(t *testing.T) {
		me := new(mocks2.MockEncoder)

		// Create an instance of ocr3 plugin
		plugin := &ocr3Plugin{
			Logger:        log.New(io.Discard, "", 0),
			ReportEncoder: me,
			Config: config.OffchainConfig{
				GasOverheadPerUpkeep: 100_000,
				GasLimitPerReport:    5_000_000,
				MaxUpkeepBatchSize:   1,
			},
		}

		outcome := ocr2keepersv3.AutomationOutcome{
			BasicOutcome: ocr2keepersv3.BasicOutcome{
				Performable: []ocr2keepers.CheckResult{
					{
						UpkeepID: ocr2keepers.UpkeepIdentifier([32]byte{1}),
						Trigger: ocr2keepers.Trigger{
							BlockNumber: 4,
							BlockHash:   [32]byte{0},
							LogTriggerExtension: &ocr2keepers.LogTriggerExtension{
								TxHash: [32]byte{1},
								Index:  4,
							},
						},
						GasAllocated: 5_000_000,
						PerformData:  []byte(`0x`),
					},
				},
			},
		}

		// me.On("Encode", mock.Anything).Return([]byte(``), nil)

		rawOutcome, err := outcome.Encode()
		assert.NoError(t, err, "no error during encoding")

		reports, err := plugin.Reports(2, rawOutcome)
		assert.NoError(t, err, "no error from generating reports")
		assert.Len(t, reports, 0, "report length should be 0")

		me.AssertExpectations(t)
	})
}
