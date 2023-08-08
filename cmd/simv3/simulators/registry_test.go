package simulators

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/smartcontractkit/ocr2keepers/pkg/encoding"
	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg/v3/types"
)

func TestCheckUpkeep(t *testing.T) {
	t.Skip()

	tel := new(MockRPCTelemetry)
	mct := new(MockContractTelemetry)

	type enc struct {
		SimulatedReportEncoder
		encoding.BasicEncoder
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
				Performs: map[string]ocr2keepers.TransmitEvent{
					"7": {
						WorkID:     "4|20",
						UpkeepID:   ocr2keepers.UpkeepIdentifier([32]byte{20}),
						CheckBlock: ocr2keepers.BlockNumber(4),
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

	payload1 := ocr2keepers.UpkeepPayload{
		Upkeep: ocr2keepers.ConfiguredUpkeep{
			ID: ocr2keepers.UpkeepIdentifier([32]byte{201}),
		},
		Trigger: ocr2keepers.NewTrigger(8, [32]byte{1, 2, 3}),
	}
	// generateID was deprecated; find new way to create id
	// payload1.ID = payload1.GenerateID()

	res, err := contract.CheckUpkeeps(context.Background(), payload1)
	assert.NoError(t, err)
	assert.Len(t, res, 1)

	// TODO: fix the following tests
	//assert.Equal(t, checkKey, res[0].Key)
	//assert.Equal(t, types.NotEligible, res[0].State)

	tel.On("RegisterCall", "checkUpkeep", mock.Anything, nil)

	payload2 := ocr2keepers.UpkeepPayload{
		Upkeep: ocr2keepers.ConfiguredUpkeep{
			ID: ocr2keepers.UpkeepIdentifier([32]byte{201}),
		},
		Trigger: ocr2keepers.NewTrigger(11, [32]byte{1, 2, 3}),
	}
	// generateID was deprecated; find new way to create id
	// payload2.ID = payload2.GenerateID()

	res, err = contract.CheckUpkeeps(context.Background(), payload2)
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

func (_m *MockContractTelemetry) CheckID(id string, block ocr2keepers.BlockKey) {
	_m.Mock.Called(id, block)
}
