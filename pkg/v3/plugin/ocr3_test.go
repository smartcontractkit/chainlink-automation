package plugin

/*
func TestObservation(t *testing.T) {
	// Create an instance of ocr3 plugin
	plugin := &ocr3Plugin[int]{
		Logger: log.New(io.Discard, "", 0),
	}

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
	expectedEncodedOutcome := []byte(`{"Instructions":null,"Metadata":null,"Performable":null}`)
	assert.Equal(t, types.Observation(expectedEncodedOutcome), observation)
}

func TestOcr3Plugin_Outcome(t *testing.T) {
	t.Run("malformed observations returns an error", func(t *testing.T) {
		// Create an instance of ocr3 plugin
		plugin := &ocr3Plugin[int]{
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

	t.Run("given three observations, in which two are identical, one observations is added to the outcome", func(t *testing.T) {
		// Create an instance of ocr3 plugin
		plugin := &ocr3Plugin[int]{
			Logger: log.New(io.Discard, "", 0),
		}

		// Create a sample outcome for decoding
		outcomeContext := ocr3types.OutcomeContext{
			PreviousOutcome: []byte(`{"Instructions":["instruction1"],"Metadata":{"key":"value"},"Performable":[]}`),
		}

		automationObservation1 := ocr2keepersv3.AutomationObservation{
			Performable: []ocr2keepers.CheckResult{
				{
					Eligible:  true,
					Retryable: false,
					Payload: ocr2keepers.UpkeepPayload{
						ID: "123",
						Upkeep: ocr2keepers.ConfiguredUpkeep{
							ID:   ocr2keepers.UpkeepIdentifier("456"),
							Type: 1,
						},
						Trigger: ocr2keepers.Trigger{
							BlockNumber: 987,
							BlockHash:   "789",
							Extension:   333,
						},
					},
				},
			},
		}
		automationObservation2 := ocr2keepersv3.AutomationObservation{
			Performable: []ocr2keepers.CheckResult{
				{
					Eligible:  true,
					Retryable: false,
					Payload: ocr2keepers.UpkeepPayload{
						ID: "123",
						Upkeep: ocr2keepers.ConfiguredUpkeep{
							ID:   ocr2keepers.UpkeepIdentifier("456"),
							Type: 1,
						},
						Trigger: ocr2keepers.Trigger{
							BlockNumber: 987,
							BlockHash:   "789",
							Extension:   333,
						},
					},
				},
			},
		}
		automationObservation3 := ocr2keepersv3.AutomationObservation{
			Performable: []ocr2keepers.CheckResult{
				{
					Eligible:  true,
					Retryable: false,
					Payload: ocr2keepers.UpkeepPayload{
						ID: "112233",
						Upkeep: ocr2keepers.ConfiguredUpkeep{
							ID:   ocr2keepers.UpkeepIdentifier("456"),
							Type: 1,
						},
						Trigger: ocr2keepers.Trigger{
							BlockNumber: 987,
							BlockHash:   "789",
							Extension:   333,
						},
					},
				},
			},
		}

		obs1, err := automationObservation1.Encode()
		assert.NoError(t, err)
		obs2, err := automationObservation2.Encode()
		assert.NoError(t, err)
		obs3, err := automationObservation3.Encode()
		assert.NoError(t, err)

		observations := []types.AttributedObservation{
			{
				Observation: obs1,
			},
			{
				Observation: obs2, // payload matches obs1, giving 2 counts of the same payload
			},
			{
				Observation: obs3, // this single report instance won't reach the quorum threshold
			},
		}

		outcome, err := plugin.Outcome(outcomeContext, nil, observations)
		assert.NoError(t, err)

		automationOutcome, err := ocr2keepersv3.DecodeAutomationOutcome(outcome)
		assert.NoError(t, err)

		// obs1 and obs2 are identical, so they will be considered the same result. obs3 doesn't reach the quorum threshold
		assert.Len(t, automationOutcome.Performable, 1)
	})
}
*/
