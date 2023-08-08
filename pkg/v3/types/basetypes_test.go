package types

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTriggerUnmarshal(t *testing.T) {
	input := Trigger{
		BlockNumber: 5,
		BlockHash:   [32]byte{1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4},
		LogTriggerExtension: &LogTriggerExtenstion{
			LogTxHash: [32]byte{1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4},
			Index:     99,
		},
	}

	encoded, _ := json.Marshal(input)

	rawJSON := `{"BlockNumber":5,"BlockHash":[1,2,3,4,1,2,3,4,1,2,3,4,1,2,3,4,1,2,3,4,1,2,3,4,1,2,3,4,1,2,3,4],"LogTriggerExtension":{"LogTxHash":[1,2,3,4,1,2,3,4,1,2,3,4,1,2,3,4,1,2,3,4,1,2,3,4,1,2,3,4,1,2,3,4],"Index":99,"BlockHash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"BlockNumber":0}}`

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
		BlockHash:   [32]byte{1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4},
		LogTriggerExtension: &LogTriggerExtenstion{
			LogTxHash: [32]byte{1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4},
			Index:     99,
		},
	}

	assert.Equal(t, expected, output, "decoding should leave extension in its raw encoded state")
}

func TestTriggerUnmarshal_EmptyExtension(t *testing.T) {
	input := Trigger{
		BlockNumber: 5,
		BlockHash:   [32]byte{1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4},
	}

	encoded, _ := json.Marshal(input)

	rawJSON := `{"BlockNumber":5,"BlockHash":[1,2,3,4,1,2,3,4,1,2,3,4,1,2,3,4,1,2,3,4,1,2,3,4,1,2,3,4,1,2,3,4],"LogTriggerExtension":null}`

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
		BlockHash:   [32]byte{1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4},
	}

	assert.Equal(t, expected, output, "decoding should leave extension in its raw encoded state")
}

func TestUpkeepIdentifier_BigInt(t *testing.T) {
	tests := []struct {
		name       string
		id         *big.Int
		wantHex    string
		upkeepType UpkeepType
	}{
		{
			name: "log trigger from decimal",
			id: func() *big.Int {
				id, _ := big.NewInt(0).SetString("32329108151019397958065800113404894502874153543356521479058624064899121404671", 10)
				return id
			}(),
			wantHex: "32329108151019397958065800113404894502874153543356521479058624064899121404671",
		},
		{
			name: "condition trigger from hex",
			id: func() *big.Int {
				id, _ := big.NewInt(0).SetString("4779a07400000000000000000000000042d780684c0bbe59fab87e6ea7f3daff", 16)
				return id
			}(),
			wantHex: "32329108151019397958065800113404894502533871176435583015595249457467353193215",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			uid := new(UpkeepIdentifier)
			uid.FromBigInt(tc.id)
			assert.Equal(t, tc.wantHex, uid.String())
			assert.Equal(t, tc.id, uid.BigInt())
		})
	}
}
