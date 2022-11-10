package simulators

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/smartcontractkit/ocr2keepers/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestCheckUpkeep(t *testing.T) {
	tel := new(MockRPCTelemetry)
	mct := new(MockContractTelemetry)
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
				Performs: map[string]types.PerformLog{
					"7": {
						Key: types.UpkeepKey([]byte("4|20")),
					},
				},
			},
		},
		rpc:       rpc,
		telemetry: mct,
	}

	tel.On("RegisterCall", "checkUpkeep", mock.Anything, nil)
	tel.On("AddRateDataPoint", mock.Anything)
	tel.On("RegisterCall", "simulatePerform", mock.Anything, nil)
	tel.On("AddRateDataPoint", mock.Anything)

	mct.On("CheckKey", mock.Anything)

	checkKey := types.UpkeepKey([]byte("8|201"))
	ok, res, err := contract.CheckUpkeep(context.Background(), checkKey)

	assert.Equal(t, false, ok)
	assert.NoError(t, err)
	assert.Equal(t, checkKey, res.Key)
	assert.Equal(t, types.NotEligible, res.State)

	tel.On("RegisterCall", "checkUpkeep", mock.Anything, nil)
	checkKey2 := types.UpkeepKey([]byte("11|201"))
	ok, res, err = contract.CheckUpkeep(context.Background(), checkKey2)

	assert.Equal(t, true, ok)
	assert.NoError(t, err)
	assert.Equal(t, checkKey2, res.Key)
	assert.Equal(t, types.Eligible, res.State)
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

func (_m *MockContractTelemetry) CheckKey(key []byte) {
	_m.Mock.Called(key)
}
