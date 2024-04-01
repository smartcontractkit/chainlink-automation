package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDecodeOffchainConfig(t *testing.T) {
	for _, tc := range []struct {
		Name              string
		EncodedData       []byte
		ExpectedErrString string
		ExpectedConfig    OffchainConfig
	}{
		{
			Name: "Happy path",
			EncodedData: []byte(`
				{
					"performLockoutWindow": 1000,
					"targetProbability": "0.999",
					"targetInRounds": 1,
					"samplingJobDuration": 1000,
					"minConfirmations": 10,
					"gasLimitPerReport": 10,
					"gasOverheadPerUpkeep": 100,
					"maxUpkeepBatchSize": 100,
					"reportBlockLag": 100,
					"mercuryLookup": true,
					"logProviderConfig": {
						"blockRate": 32,
						"logLimit": 50
					}
				}
			`),
			ExpectedErrString: "",
			ExpectedConfig: OffchainConfig{
				PerformLockoutWindow: 1000,
				TargetProbability:    "0.999",
				TargetInRounds:       1,
				MinConfirmations:     10,
				GasLimitPerReport:    10,
				GasOverheadPerUpkeep: 100,
				MaxUpkeepBatchSize:   100,
				LogProviderConfig: LogProviderConfig{
					BlockRate: 32,
					LogLimit:  50,
				},
			},
		},
		{
			Name: "Extra field uniqueReports",
			EncodedData: []byte(`
				{
					"performLockoutWindow": 1000,
					"uniqueReports": true,
					"targetProbability": "0.999",
					"targetInRounds": 1,
					"samplingJobDuration": 1000,
					"minConfirmations": 10,
					"gasLimitPerReport": 10,
					"gasOverheadPerUpkeep": 100,
					"maxUpkeepBatchSize": 100,
					"logProviderConfig": {
						"blockRate": 10,
						"logLimit": 20,
						"additionalField": 30
					}
				}
			`),
			ExpectedErrString: "",
			ExpectedConfig: OffchainConfig{
				PerformLockoutWindow: 1000,
				TargetProbability:    "0.999",
				TargetInRounds:       1,
				MinConfirmations:     10,
				GasLimitPerReport:    10,
				GasOverheadPerUpkeep: 100,
				MaxUpkeepBatchSize:   100,
				LogProviderConfig: LogProviderConfig{
					BlockRate: 10,
					LogLimit:  20,
				},
			},
		},
		{
			Name:              "Missing values",
			EncodedData:       []byte(`{}`),
			ExpectedErrString: "",
			ExpectedConfig: OffchainConfig{
				PerformLockoutWindow: 1200000,
				TargetProbability:    "0.99999",
				TargetInRounds:       1,
				MinConfirmations:     0,
				GasLimitPerReport:    5_300_000,
				GasOverheadPerUpkeep: 300_000,
				MaxUpkeepBatchSize:   1,
				LogProviderConfig: LogProviderConfig{
					BlockRate: 0,
					LogLimit:  0,
				},
			},
		},
		{
			Name: "Negative values",
			EncodedData: []byte(`
				{
					"performLockoutWindow": -1000,
					"targetProbability": "0.999",
					"targetInRounds": -1,
					"samplingJobDuration": -1000,
					"minConfirmations": -10,
					"gasLimitPerReport": 0,
					"gasOverheadPerUpkeep": 0,
					"maxUpkeepBatchSize": -100,
					"reportBlockLag": -100,
					"logProviderConfig": {
						"blockRate": 0,
						"logLimit": 0
					}
				}
			`),
			ExpectedErrString: "",
			ExpectedConfig: OffchainConfig{
				PerformLockoutWindow: 1200000,
				TargetProbability:    "0.999",
				TargetInRounds:       1,
				MinConfirmations:     0,
				GasLimitPerReport:    5_300_000,
				GasOverheadPerUpkeep: 300_000,
				MaxUpkeepBatchSize:   1,
				LogProviderConfig: LogProviderConfig{
					BlockRate: 0,
					LogLimit:  0,
				},
			},
		},
		{
			Name: "Unexpected type",
			EncodedData: []byte(`
				{
					"performLockoutWindow": "string",
					"targetProbability": "0.999",
					"targetInRounds": -1,
					"samplingJobDuration": -1000,
					"minConfirmations": -10,
					"gasLimitPerReport": 0,
					"gasOverheadPerUpkeep": 0,
					"maxUpkeepBatchSize": -100,
					"logProviderConfig": {
						"numOfLogUpkeeps": false,
						"logLimitHigh": true
					}
				}
			`),
			ExpectedErrString: "json: cannot unmarshal string",
			ExpectedConfig:    OffchainConfig{},
		},
	} {
		t.Run(tc.Name, func(t *testing.T) {
			config, err := DecodeOffchainConfig(tc.EncodedData)
			if tc.ExpectedErrString != "" {
				assert.ErrorContains(t, err, tc.ExpectedErrString)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, config, tc.ExpectedConfig)
			}
		})
	}
}
