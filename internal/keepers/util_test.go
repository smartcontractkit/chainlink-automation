package keepers

import (
	"sort"
	"testing"

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
