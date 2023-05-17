package simulators

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg"
	"github.com/smartcontractkit/ocr2keepers/pkg/encoding"
)

func TestCheckUpkeep(t *testing.T) {
	tel := new(MockRPCTelemetry)
	mct := new(MockContractTelemetry)

	type enc struct {
		SimulatedReportEncoder
		encoding.KeyBuilder
	}

	rpc := NewSimulatedRPC(0, 1000, 0, tel)
	contract := &SimulatedContract{
		avgLatency: 2,
		upkeeps: map[string]SimulatedUpkeep{
			"201": {
				ID: big.NewInt(201),
				EligibleAt: []*big.Int{
					big.NewInt(5),
					big.NewInt(10),
					big.NewInt(15),
					big.NewInt(20),
				},
				Performs: map[string]ocr2keepers.PerformLog{
					"7": {
						Key: ocr2keepers.UpkeepKey([]byte("4|20")),
					},
				},
			},
		},
		enc:       enc{},
		rpc:       rpc,
		telemetry: mct,
	}

	tel.On("RegisterCall", "checkUpkeep", mock.Anything, nil)
	tel.On("AddRateDataPoint", mock.Anything)
	tel.On("RegisterCall", "simulatePerform", mock.Anything, nil)
	tel.On("AddRateDataPoint", mock.Anything)

	mct.On("CheckKey", mock.Anything)

	checkKey := ocr2keepers.UpkeepKey([]byte("8|201"))
	res, err := contract.CheckUpkeep(context.Background(), false, checkKey)
	assert.NoError(t, err)
	assert.Len(t, res, 1)

	// TODO: fix the following tests
	//assert.Equal(t, checkKey, res[0].Key)
	//assert.Equal(t, types.NotEligible, res[0].State)

	tel.On("RegisterCall", "checkUpkeep", mock.Anything, nil)
	checkKey2 := ocr2keepers.UpkeepKey([]byte("11|201"))
	res, err = contract.CheckUpkeep(context.Background(), false, checkKey2)
	assert.NoError(t, err)
	assert.Len(t, res, 1)
	//assert.Equal(t, checkKey2, res[0].Key)
	//assert.Equal(t, types.Eligible, res[0].State)
}

type MockRPCTelemetry struct {
	mock.Mock
}

func (_m *MockRPCTelemetry) RegisterCall(name string, t time.Duration, err error) {
	_m.Mock.Called(name, t, err)
}

func (_m *MockRPCTelemetry) AddRateDataPoint(p int) {
	_m.Mock.Called(p)
}

type MockContractTelemetry struct {
	mock.Mock
}

func (_m *MockContractTelemetry) CheckKey(key ocr2keepers.UpkeepKey) {
	_m.Mock.Called(key)
}
