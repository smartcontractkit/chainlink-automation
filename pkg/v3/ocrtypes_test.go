package ocr2keepers

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"

	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg"
)

func TestAutomationObservation_Encode(t *testing.T) {
	observation := AutomationObservation{
		Instructions: []string{"instruction1", "instruction2"},
		Metadata:     map[string]interface{}{"key": "value"},
		Performable: []ocr2keepers.CheckResult{
			{
				Payload: ocr2keepers.UpkeepPayload{
					ID: "abc",
				},
				Retryable: false,
				Eligible:  false,
			},
		},
	}

	expectedJSON := `{"Instructions":["instruction1","instruction2"],"Metadata":{"key":"value"},"Performable":[{"Payload":{"ID":"abc"},"Retryable":false, "Eligible":false}]}`

	data, err := observation.Encode()
	assert.NoError(t, err)

	var encoded map[string]interface{}
	err = json.Unmarshal(data, &encoded)
	assert.NoError(t, err)

	var expected map[string]interface{}
	err = json.Unmarshal([]byte(expectedJSON), &expected)
	assert.NoError(t, err)

	assert.True(t, reflect.DeepEqual(encoded, expected), "Encoded data does not match the expected JSON. Expected: %v, Got: %v", expected, encoded)
}

func TestAutomationObservation_Decode(t *testing.T) {
	jsonData := `{"Instructions":["instruction1","instruction2"],"Metadata":{"key":"value"},"Performable":[{}]}`

	expectedObservation := AutomationObservation{
		Instructions: []string{"instruction1", "instruction2"},
		Metadata:     map[string]interface{}{"key": "value"},
		Performable:  []ocr2keepers.CheckResult{{}},
	}

	data := []byte(jsonData)

	observation, err := DecodeAutomationObservation(data)
	assert.NoError(t, err)

	assert.True(t, reflect.DeepEqual(observation, expectedObservation), "Decoded observation does not match the expected value. Expected: %v, Got: %v", expectedObservation, observation)
}

func TestAutomationOutcome_Encode(t *testing.T) {
	outcome := AutomationOutcome{
		Instructions: []string{"instruction1", "instruction2"},
		Metadata:     map[string]interface{}{"key": "value"},
		Performable: []ocr2keepers.CheckResult{
			{
				Payload: ocr2keepers.UpkeepPayload{
					ID: "abc",
				},
				Retryable: false,
				Eligible:  false,
			},
		},
	}

	expectedJSON := `{"Instructions":["instruction1","instruction2"],"Metadata":{"key":"value"},"Performable":[{"Payload":{"ID":"abc"},"Retryable":false, "Eligible":false}]}`

	data, err := outcome.Encode()
	assert.NoError(t, err)

	var encoded map[string]interface{}
	err = json.Unmarshal(data, &encoded)
	assert.NoError(t, err)

	var expected map[string]interface{}
	err = json.Unmarshal([]byte(expectedJSON), &expected)
	assert.NoError(t, err)

	assert.True(t, reflect.DeepEqual(encoded, expected), "Encoded data does not match the expected JSON. Expected: %v, Got: %v", expected, encoded)
}

func TestAutomationOutcome_Decode(t *testing.T) {
	jsonData := `{"Instructions":["instruction1","instruction2"],"Metadata":{"key":"value"},"Performable":[{}]}`

	expectedOutcome := AutomationOutcome{
		Instructions: []string{"instruction1", "instruction2"},
		Metadata:     map[string]interface{}{"key": "value"},
		Performable:  []ocr2keepers.CheckResult{{}},
	}

	data := []byte(jsonData)

	outcome, err := DecodeAutomationOutcome(data)
	assert.NoError(t, err)

	assert.True(t, reflect.DeepEqual(outcome, expectedOutcome), "Decoded outcome does not match the expected value. Expected: %v, Got: %v", expectedOutcome, outcome)
}
