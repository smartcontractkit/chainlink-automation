package types

import (
	"math/big"
	"testing"

	"github.com/pkg/errors"
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
			"reportBlockLag": 100,
			"mercuryLookup": true
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
			MercuryLookup:        true,
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
			MercuryLookup:        false,
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
			MercuryLookup:        false,
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
			MercuryLookup:        false,
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

func TestOffchainConfig_Encode(t *testing.T) {
	t.Run("unmarshall-able config causes a panic", func(t *testing.T) {
		oldMarshalFn := marshalFn
		marshalFn = func(v any) ([]byte, error) {
			return nil, errors.New("panicking")
		}
		defer func() {
			marshalFn = oldMarshalFn
		}()

		defer func() {
			if r := recover(); r == nil {
				t.Fatalf("expected a panic, but didn't panic")
			}
		}()

		config := OffchainConfig{
			PerformLockoutWindow: 1,
		}

		config.Encode()
	})

	t.Run("config is marshalled into json bytes", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("unexpected panic")
			}
		}()

		config := OffchainConfig{
			PerformLockoutWindow: 1,
			MercuryLookup:        true,
		}

		bytes := config.Encode()
		assert.Equal(t, bytes, []byte(`{"performLockoutWindow":1,"targetProbability":"","targetInRounds":0,"samplingJobDuration":0,"minConfirmations":0,"gasLimitPerReport":0,"gasOverheadPerUpkeep":0,"maxUpkeepBatchSize":0,"reportBlockLag":0,"mercuryLookup":true}`))
	})
}

func TestUpkeepIdentifier_BigInt(t *testing.T) {
	id := UpkeepIdentifier("123")
	idInt, ok := id.BigInt()
	if !ok {
		t.Fatalf("unexpected identifier")
	}
	assert.Equal(t, idInt, big.NewInt(123))
}
