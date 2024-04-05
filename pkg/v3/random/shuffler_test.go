package random

import (
	"github.com/smartcontractkit/chainlink-common/pkg/types/automation"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestShuffler_Shuffle(t *testing.T) {
	shuffler := Shuffler{Source: rand.NewSource(0)}
	arr := []automation.UpkeepPayload{
		{WorkID: "1"},
		{WorkID: "2"},
		{WorkID: "3"},
		{WorkID: "4"},
		{WorkID: "5"},
	}
	arr = shuffler.Shuffle(arr)
	assert.Equal(t, arr, []automation.UpkeepPayload{
		{WorkID: "3"},
		{WorkID: "4"},
		{WorkID: "2"},
		{WorkID: "1"},
		{WorkID: "5"}},
	)

	// Sorting again using a used shuffler should yield a different result
	arr = []automation.UpkeepPayload{
		{WorkID: "1"},
		{WorkID: "2"},
		{WorkID: "3"},
		{WorkID: "4"},
		{WorkID: "5"},
	}
	arr = shuffler.Shuffle(arr)
	assert.Equal(t, arr, []automation.UpkeepPayload{
		{WorkID: "3"},
		{WorkID: "4"},
		{WorkID: "1"},
		{WorkID: "5"},
		{WorkID: "2"}},
	)

	// Sorting again using a new shuffler with the same pseudo-random source should yield the same result
	shuffler2 := Shuffler{Source: rand.NewSource(0)}
	arr2 := []automation.UpkeepPayload{
		{WorkID: "1"},
		{WorkID: "2"},
		{WorkID: "3"},
		{WorkID: "4"},
		{WorkID: "5"},
	}

	arr2 = shuffler2.Shuffle(arr2)
	assert.Equal(t, arr2, []automation.UpkeepPayload{
		{WorkID: "3"},
		{WorkID: "4"},
		{WorkID: "2"},
		{WorkID: "1"},
		{WorkID: "5"}},
	)
}

func TestShuffler_ShuffleString(t *testing.T) {
	assert.Equal(t, ShuffleString("12345", [16]byte{0}), "14523")
	// ShuffleString should be deterministic based on rSrc
	assert.Equal(t, ShuffleString("12345", [16]byte{0}), "14523")
	assert.Equal(t, ShuffleString("12345", [16]byte{1}), "51243")
	assert.Equal(t, ShuffleString("123456", [16]byte{0}), "516423")
	assert.Equal(t, ShuffleString("", [16]byte{0}), "")
	assert.Equal(t, ShuffleString("dsv$\u271387csdv0-`", [16]byte{0}), "d0v`$-8vsscd\u27137")
}
