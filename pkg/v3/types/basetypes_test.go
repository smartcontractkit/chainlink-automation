package types

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTriggerUnmarshal(t *testing.T) {
	input := Trigger{
		BlockNumber: 5,
		BlockHash:   [32]byte{1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4},
		LogTriggerExtension: &LogTriggerExtension{
			TxHash: [32]byte{1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4},
			Index:  99,
		},
	}

	encoded, _ := json.Marshal(input)

	rawJSON := `{"BlockNumber":5,"BlockHash":[1,2,3,4,1,2,3,4,1,2,3,4,1,2,3,4,1,2,3,4,1,2,3,4,1,2,3,4,1,2,3,4],"LogTriggerExtension":{"TxHash":[1,2,3,4,1,2,3,4,1,2,3,4,1,2,3,4,1,2,3,4,1,2,3,4,1,2,3,4,1,2,3,4],"Index":99,"BlockHash":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"BlockNumber":0}}`

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
		LogTriggerExtension: &LogTriggerExtension{
			TxHash: [32]byte{1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4},
			Index:  99,
		},
	}

	assert.Equal(t, expected, output, "decoding should leave extension in its raw encoded state")
}

func TestTriggerString(t *testing.T) {
	input := Trigger{
		BlockNumber: 5,
		BlockHash:   [32]byte{1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4},
		LogTriggerExtension: &LogTriggerExtension{
			TxHash: [32]byte{1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4},
			Index:  99,
		},
	}

	stringified := fmt.Sprintf("%v", input)
	expected := `
		{
			"BlockNumber":5,
			"BlockHash":"0102030401020304010203040102030401020304010203040102030401020304",
			"LogTriggerExtension": {
				"BlockHash":"0000000000000000000000000000000000000000000000000000000000000000",
				"BlockNumber":0,
				"Index":99,
				"TxHash":"0102030401020304010203040102030401020304010203040102030401020304"
			}
		}`

	assertJSONEqual(t, expected, stringified)

	input = Trigger{
		BlockNumber: 5,
		BlockHash:   [32]byte{1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4},
	}

	stringified = fmt.Sprintf("%v", input)
	expected = `{"BlockNumber":5,"BlockHash":"0102030401020304010203040102030401020304010203040102030401020304"}`

	assertJSONEqual(t, expected, stringified)
}

func TestLogIdentifier(t *testing.T) {
	input := Trigger{
		BlockNumber: 5,
		BlockHash:   [32]byte{1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4},
		LogTriggerExtension: &LogTriggerExtension{
			TxHash: [32]byte{1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4},
			Index:  99,
		},
	}

	logIdentifier := input.LogTriggerExtension.LogIdentifier()
	assert.Equal(t, hex.EncodeToString(logIdentifier), "010203040102030401020304010203040102030401020304010203040102030400000063")
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
		name          string
		id            *big.Int
		want          string
		ignoreConvert bool
	}{
		{
			name: "log trigger from decimal",
			id: func() *big.Int {
				id, _ := big.NewInt(0).SetString("32329108151019397958065800113404894502874153543356521479058624064899121404671", 10)
				return id
			}(),
			want: "32329108151019397958065800113404894502874153543356521479058624064899121404671",
		},
		{
			name: "condition trigger from hex",
			id: func() *big.Int {
				id, _ := big.NewInt(0).SetString("4779a07400000000000000000000000042d780684c0bbe59fab87e6ea7f3daff", 16)
				return id
			}(),
			want: "32329108151019397958065800113404894502533871176435583015595249457467353193215",
		},
		{
			name: "0 upkeep ID",
			id:   big.NewInt(0),
			want: "0",
		},
		{
			name: "random upkeep ID",
			id: func() *big.Int {
				id, _ := big.NewInt(0).SetString("32329108151019423423423", 10)
				return id
			}(),
			want: "32329108151019423423423",
		},
		{
			name:          "negative upkeep ID",
			id:            big.NewInt(-10),
			want:          "0",
			ignoreConvert: true,
		},
		{
			name: "max upkeep ID (2^256-1)",
			id: func() *big.Int {
				id, _ := big.NewInt(0).SetString("115792089237316195423570985008687907853269984665640564039457584007913129639935", 10)
				return id
			}(),
			want: "115792089237316195423570985008687907853269984665640564039457584007913129639935",
		},
		{
			name: "out of range upkeep ID (2^256)",
			id: func() *big.Int {
				id, _ := big.NewInt(0).SetString("115792089237316195423570985008687907853269984665640564039457584007913129639936", 10)
				return id
			}(),
			want:          "0",
			ignoreConvert: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			uid := new(UpkeepIdentifier)
			ok := uid.FromBigInt(tc.id)
			assert.Equal(t, !tc.ignoreConvert, ok)
			assert.Equal(t, tc.want, uid.String())
			if !tc.ignoreConvert {
				assert.Equal(t, tc.id.String(), uid.BigInt().String())
			}
		})
	}
}

func TestCheckResultString(t *testing.T) {
	input := CheckResult{
		PipelineExecutionState: 1,
		Retryable:              true,
		Eligible:               true,
		IneligibilityReason:    10,
		UpkeepID:               UpkeepIdentifier{1, 2, 3, 4, 5, 6, 7, 8},
		Trigger: Trigger{
			BlockNumber: 5,
			BlockHash:   [32]byte{1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4},
			LogTriggerExtension: &LogTriggerExtension{
				TxHash: [32]byte{1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4, 1, 2, 3, 4},
				Index:  99,
			},
		},
		WorkID:       "work id",
		GasAllocated: 1001,
		PerformData:  []byte{1, 2, 3, 4, 5, 6},
		FastGasWei:   big.NewInt(12),
		LinkNative:   big.NewInt(13),
	}

	result := fmt.Sprintf("%v", input)
	expected := `
		{
			"PipelineExecutionState":1,
			"Retryable":true,
			"Eligible":true,
			"IneligibilityReason":10,
			"UpkeepID":455867356320691211288303676705517652851520854420902457558325773249309310976,
			"Trigger": {
				"BlockHash":"0102030401020304010203040102030401020304010203040102030401020304",
				"BlockNumber":5,
				"LogTriggerExtension": {
					"BlockHash":"0000000000000000000000000000000000000000000000000000000000000000",
					"BlockNumber":0,
					"Index":99,
					"TxHash":"0102030401020304010203040102030401020304010203040102030401020304"
				}
			},
			"WorkID":"work id",
			"GasAllocated":1001,
			"PerformData":"010203040506",
			"FastGasWei":12,
			"LinkNative":13
		}
	`
	assertJSONEqual(t, expected, result)
	assertJSONContainsAllStructFields(t, result, input)
}

func assertJSONEqual(t *testing.T, expected, actual string) {
	var expectedMap, actualMap map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(expected), &expectedMap), "expected is invalid json")
	require.NoError(t, json.Unmarshal([]byte(actual), &actualMap), "actual is invalid json")
	assert.True(t, reflect.DeepEqual(expectedMap, actualMap), "expected and result json strings do not match")
}

func assertJSONContainsAllStructFields(t *testing.T, jsonString string, anyStruct interface{}) {
	// if fields are added to the struct in the future, but omitted from the "pretty" string template, this test will fail
	var jsonMap map[string]interface{}
	var structMap map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(jsonString), &jsonMap), "jsonString is invalid json")
	structJson, err := json.Marshal(anyStruct)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(structJson, &structMap))
	assertCongruentKeyStructure(t, structMap, jsonMap)
}

func assertCongruentKeyStructure(t *testing.T, structMap, jsonMap map[string]interface{}) {
	// this functions asserts that the two inputs have congruent key shapes, while disregarding
	// the values
	for k := range structMap {
		assert.True(t, jsonMap[k] != nil, "json string does not contain field %s", k)
		if nested1, ok := structMap[k].(map[string]interface{}); ok {
			if nested2, ok := jsonMap[k].(map[string]interface{}); ok {
				assertCongruentKeyStructure(t, nested1, nested2)
			} else {
				assert.Fail(t, "maps do not contain the same type for key %s", k)
			}
		}
	}
}
