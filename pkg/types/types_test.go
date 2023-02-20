package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDecode(t *testing.T) {
	tests := []struct {
		Name              string
		EncodedData       []byte
		ExpectedErrString string
		ExpectedConfig    OffchainConfig
	}{
		{Name: "Happy path", EncodedData: []byte(`
		{
			"performLockoutWindow": 1000,
			"targetProbability": "0.999",
			"targetInRounds": 1,
			"samplingJobDuration": 1000,
			"minConfirmations": 10,
			"gasLimitPerReport": 10,
			"gasOverheadPerUpkeep": 100,
			"maxUpkeepBatchSize": 100,
			"reportBlockLag": 100
		}
	`), ExpectedErrString: "", ExpectedConfig: OffchainConfig{
			PerformLockoutWindow: 1000,
			TargetProbability:    "0.999",
			TargetInRounds:       1,
			SamplingJobDuration:  1000,
			MinConfirmations:     10,
			GasLimitPerReport:    10,
			GasOverheadPerUpkeep: 100,
			MaxUpkeepBatchSize:   100,
			ReportBlockLag:       100,
		}},

		{Name: "Extra field uniqueReports", EncodedData: []byte(`
		{
			"performLockoutWindow": 1000,
			"uniqueReports": true,
			"targetProbability": "0.999",
			"targetInRounds": 1,
			"samplingJobDuration": 1000,
			"minConfirmations": 10,
			"gasLimitPerReport": 10,
			"gasOverheadPerUpkeep": 100,
			"maxUpkeepBatchSize": 100
		}
	`), ExpectedErrString: "", ExpectedConfig: OffchainConfig{
			PerformLockoutWindow: 1000,
			TargetProbability:    "0.999",
			TargetInRounds:       1,
			SamplingJobDuration:  1000,
			MinConfirmations:     10,
			GasLimitPerReport:    10,
			GasOverheadPerUpkeep: 100,
			MaxUpkeepBatchSize:   100,
		}},

		{Name: "Missing values", EncodedData: []byte(`
		{
		}
	`), ExpectedErrString: "", ExpectedConfig: OffchainConfig{
			PerformLockoutWindow: 1200000,
			TargetProbability:    "0.99999",
			TargetInRounds:       1,
			SamplingJobDuration:  3000,
			MinConfirmations:     0,
			GasLimitPerReport:    5_300_000,
			GasOverheadPerUpkeep: 300_000,
			MaxUpkeepBatchSize:   1,
			ReportBlockLag:       0,
		}},

		{Name: "Negative values", EncodedData: []byte(`
		{
			"performLockoutWindow": -1000,
			"targetProbability": "0.999",
			"targetInRounds": -1,
			"samplingJobDuration": -1000,
			"minConfirmations": -10,
			"gasLimitPerReport": 0,
			"gasOverheadPerUpkeep": 0,
			"maxUpkeepBatchSize": -100,
			"reportBlockLag": -100
		}
	`), ExpectedErrString: "", ExpectedConfig: OffchainConfig{
			PerformLockoutWindow: 1200000,
			TargetProbability:    "0.999",
			TargetInRounds:       1,
			SamplingJobDuration:  3000,
			MinConfirmations:     0,
			GasLimitPerReport:    5_300_000,
			GasOverheadPerUpkeep: 300_000,
			MaxUpkeepBatchSize:   1,
			ReportBlockLag:       0,
		}},

		{Name: "Unexpected type", EncodedData: []byte(`
		{
			"performLockoutWindow": "string",
			"targetProbability": "0.999",
			"targetInRounds": -1,
			"samplingJobDuration": -1000,
			"minConfirmations": -10,
			"gasLimitPerReport": 0,
			"gasOverheadPerUpkeep": 0,
			"maxUpkeepBatchSize": -100
		}
	`), ExpectedErrString: "json: cannot unmarshal string", ExpectedConfig: OffchainConfig{}},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			config, err := DecodeOffchainConfig(test.EncodedData)
			if test.ExpectedErrString != "" {
				assert.ErrorContains(t, err, test.ExpectedErrString)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, config, test.ExpectedConfig)
			}
		})
	}
}
