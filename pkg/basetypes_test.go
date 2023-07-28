package ocr2keepers

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUpkeepPayload_GenerateID(t *testing.T) {
	payload := NewUpkeepPayload(big.NewInt(111), 1, BlockKey("4"), Trigger{
		BlockNumber: 11,
		BlockHash:   "0x11111",
		Extension:   "extension111",
	}, []byte("check-data-111"))
	assert.Equal(t, "0a73a5fd0fc265416da897fa9d08509c336c847f80236389426ef0b95506912b", payload.ID)

	t.Run("empty payload id", func(t *testing.T) {
		payload = UpkeepPayload{}
		assert.Equal(t, "20c9c9e789a8e576ba9d58b1324869aefcd92545f80a5ee3834ac29b2531a8aa", payload.GenerateID())
	})
}

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
