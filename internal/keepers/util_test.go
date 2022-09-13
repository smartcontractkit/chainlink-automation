package keepers

import (
	"fmt"
	"sort"
	"testing"

	"github.com/smartcontractkit/libocr/offchainreporting2/types"
	ktypes "github.com/smartcontractkit/ocr2keepers/pkg/types"
	"github.com/stretchr/testify/assert"
)

func TestDedupe(t *testing.T) {
	tests := []struct {
		Name           string
		Sets           [][]string
		ExpectedResult []string
		ExpectedError  error
	}{
		{Name: "Empty Matching Sets", ExpectedResult: nil, ExpectedError: ErrNotEnoughInputs},
		{Name: "Single Matching Set", Sets: [][]string{{"1", "2", "3"}}, ExpectedResult: []string{"1", "2", "3"}},
		{Name: "Double Identical", Sets: [][]string{{"1", "2", "3"}, {"1", "2", "3"}}, ExpectedResult: []string{"1", "2", "3"}},
		{Name: "Double No Match", Sets: [][]string{{"1", "2", "3"}, {"4", "5", "6"}}, ExpectedResult: []string{"1", "2", "3", "4", "5", "6"}},
		{Name: "Triple Identical", Sets: [][]string{{"1", "2", "3"}, {"1", "2", "3"}, {"1", "2", "3"}}, ExpectedResult: []string{"1", "2", "3"}},
		{Name: "Triple No Match", Sets: [][]string{{"1", "2", "3"}, {"4", "5", "6"}, {"7", "8", "9"}}, ExpectedResult: []string{"1", "2", "3", "4", "5", "6", "7", "8", "9"}},
		{Name: "Double w/ Single Overlap", Sets: [][]string{{"1", "2", "3"}, {"3", "4", "5"}}, ExpectedResult: []string{"1", "2", "3", "4", "5"}},
		{Name: "Double w/ Single Overlap and Gap", Sets: [][]string{{"1", "2", "3", "6", "7"}, {"3", "4", "5"}}, ExpectedResult: []string{"1", "2", "3", "4", "5", "6", "7"}},
		{Name: "Double w/ Double Overlap and Gap", Sets: [][]string{{"1", "2", "3", "4", "6", "7"}, {"3", "4", "5"}}, ExpectedResult: []string{"1", "2", "3", "4", "5", "6", "7"}},
		{Name: "Triple w/ Single Overlap", Sets: [][]string{{"1", "2", "3"}, {"1", "2", "3"}, {"3", "4", "5"}}, ExpectedResult: []string{"1", "2", "3", "4", "5"}},
		{Name: "Triple w/ Single Overlap and Gap", Sets: [][]string{{"1", "2", "4"}, {"1", "2", "3", "4"}, {"3", "4", "5"}}, ExpectedResult: []string{"1", "2", "3", "4", "5"}},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			var matches []string
			var err error

			matches, err = dedupe(test.Sets)

			if test.ExpectedError != nil {
				assert.ErrorIs(t, err, test.ExpectedError)
			} else {
				sort.Strings(matches)
				assert.Equal(t, test.ExpectedResult, matches)
			}
		})
	}
}

func BenchmarkSortedDedupedKeyListFunc(b *testing.B) {
	key1 := ktypes.UpkeepKey([]byte("1|1"))
	key2 := ktypes.UpkeepKey([]byte("1|2"))
	key3 := ktypes.UpkeepKey([]byte("2|1"))

	encoded := mustEncodeKeys([]ktypes.UpkeepKey{key1, key2})

	observations := []types.AttributedObservation{
		{Observation: types.Observation(encoded)},
		{Observation: types.Observation(encoded)},
		{Observation: types.Observation(encoded)},
		{Observation: types.Observation(mustEncodeKeys([]ktypes.UpkeepKey{key2, key3}))},
		{Observation: types.Observation(encoded)},
		{Observation: types.Observation(encoded)},
		{Observation: types.Observation(encoded)},
		{Observation: types.Observation(encoded)},
		{Observation: types.Observation(encoded)},
		{Observation: types.Observation(encoded)},
		{Observation: types.Observation(encoded)},
		{Observation: types.Observation(encoded)},
		{Observation: types.Observation(encoded)},
		{Observation: types.Observation(encoded)},
		{Observation: types.Observation(encoded)},
		{Observation: types.Observation(encoded)},
	}

	for i := 1; i <= 4; i++ {
		ob := observations[0 : i*4]

		b.Run(fmt.Sprintf("%d Nodes", len(ob)), func(b *testing.B) {
			b.ResetTimer()

			// run the Observation function b.N times
			for n := 0; n < b.N; n++ {

				b.StartTimer()
				_, err := sortedDedupedKeyList(ob)
				b.StopTimer()

				if err != nil {
					b.Fail()
				}
			}
		})
	}

}
