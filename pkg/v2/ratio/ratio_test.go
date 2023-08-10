package ratio

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSampleRatio_OfInt(t *testing.T) {
	tests := []struct {
		Name           string
		Ratio          float32
		Of             int
		ExpectedResult int
	}{
		{
			Name:           "30% of 100",
			Ratio:          0.3,
			Of:             100,
			ExpectedResult: 30,
		},
		{
			Name:           "33% of 10",
			Ratio:          0.33,
			Of:             10,
			ExpectedResult: 3,
		},
		{
			Name:           "Zero",
			Ratio:          0.3,
			Of:             0,
			ExpectedResult: 0,
		},
		{
			Name:           "Rounding",
			Ratio:          0.9,
			Of:             1,
			ExpectedResult: 1,
		},
		{
			Name:           "All",
			Ratio:          1.0,
			Of:             2,
			ExpectedResult: 2,
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			assert.Equal(t, test.ExpectedResult, SampleRatio(test.Ratio).OfInt(test.Of))
		})
	}
}
