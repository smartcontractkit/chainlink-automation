package types

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTriggerUnmarshal(t *testing.T) {
	input := Trigger{
		BlockNumber: 5,
		BlockHash:   "0x",
		Extension: struct {
			Key   int
			Value string
		}{
			Key:   7,
			Value: "test",
		},
	}

	encoded, _ := json.Marshal(input)

	rawJSON := `{"BlockNumber":5,"BlockHash":"0x","Extension":{"Key":7,"Value":"test"}}`

	// the encoded value above should match the rawjson expected
	assert.Equal(t, rawJSON, string(encoded), "encoded should match expected")

	// the plugin will decode and re-encode the trigger value at least once
	// before some decoding might happen
	var decodeOnce Trigger
	_ = json.Unmarshal([]byte(rawJSON), &decodeOnce)

	encoded, _ = json.Marshal(decodeOnce)

	// used the re-encoded output to verify data integrity
	var output Trigger
	err := json.Unmarshal(encoded, &output)

	assert.NoError(t, err, "no error expected from decoding")

	expected := Trigger{
		BlockNumber: 5,
		BlockHash:   "0x",
		Extension:   []byte(`{"Key":7,"Value":"test"}`),
	}

	assert.Equal(t, expected, output, "decoding should leave extension in its raw encoded state")
}

func TestTriggerUnmarshal_EmptyExtension(t *testing.T) {
	input := Trigger{
		BlockNumber: 5,
		BlockHash:   "0x",
	}

	encoded, _ := json.Marshal(input)

	rawJSON := `{"BlockNumber":5,"BlockHash":"0x","Extension":null}`

	// the encoded value above should match the rawjson expected
	assert.Equal(t, rawJSON, string(encoded), "encoded should match expected")

	// the plugin will decode and re-encode the trigger value at least once
	// before some decoding might happen
	var decodeOnce Trigger
	_ = json.Unmarshal([]byte(rawJSON), &decodeOnce)

	encoded, _ = json.Marshal(decodeOnce)

	// used the re-encoded output to verify data integrity
	var output Trigger
	err := json.Unmarshal(encoded, &output)

	assert.NoError(t, err, "no error expected from decoding")

	expected := Trigger{
		BlockNumber: 5,
		BlockHash:   "0x",
	}

	assert.Equal(t, expected, output, "decoding should leave extension in its raw encoded state")
}
