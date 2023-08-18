package config

import (
	"testing"

	"github.com/pkg/errors"
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
					"mercuryLookup": true
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
					"maxUpkeepBatchSize": 100
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
					"reportBlockLag": -100
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
					"maxUpkeepBatchSize": -100
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

func TestDecodeOffchainConfig_validator(t *testing.T) {
	oldValidators := validators
	validators = []validator{
		func(config *OffchainConfig) error {
			return errors.New("validation failure")
		},
	}
	defer func() {
		validators = oldValidators
	}()

	_, err := DecodeOffchainConfig([]byte(`
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
			"mercuryLookup": true
		}
	`))
	assert.Error(t, err)
}
