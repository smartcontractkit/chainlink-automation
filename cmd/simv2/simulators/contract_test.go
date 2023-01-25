package simulators

import (
	"context"
	"io"
	"log"
	"math/big"
	"testing"
	"time"

	"github.com/smartcontractkit/libocr/offchainreporting2/types"
	"github.com/smartcontractkit/ocr2keepers/cmd/simv2/config"
	ktypes "github.com/smartcontractkit/ocr2keepers/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestLatestConfig(t *testing.T) {
	t.Skip()
	// init mocks
	mb := new(MockBlockBroadcaster)
	md := new(MockDigester)
	me := new(MockEncoder)

	logs := log.New(io.Discard, "", 0)
	sim := NewSimulatedContract(mb, md, []SimulatedUpkeep{}, me, nil, 500, "", 0.01, 1000, nil, nil, logs)

	chBlock := make(chan config.SymBlock)

	mb.On("Subscribe", true).Return(2, chBlock)
	sim.Start()
	<-time.After(50 * time.Millisecond)

	chBlock <- config.SymBlock{
		BlockNumber: big.NewInt(5),
		Config: &types.ContractConfig{
			ConfigDigest: types.ConfigDigest([32]byte{}),
			F:            2,
		},
	}

	timer := time.NewTimer(time.Second)
	select {
	case <-sim.Notify():
		timer.Stop()
	case <-timer.C:
		assert.Fail(t, "no signal received to indicated new block")
	}

	conf, err := sim.LatestConfig(context.TODO(), 5)
	assert.NoError(t, err)
	assert.Equal(t, uint8(2), conf.F)

	mb.On("Unsubscribe", 2).Return()
	sim.Stop()
}

func TestLatestBlockHeight(t *testing.T) {
	t.Skip()
	// init mocks
	mb := new(MockBlockBroadcaster)
	md := new(MockDigester)
	me := new(MockEncoder)

	logs := log.New(io.Discard, "", 0)
	sim := NewSimulatedContract(mb, md, []SimulatedUpkeep{}, me, nil, 500, "", 0.01, 1000, nil, nil, logs)

	chBlock := make(chan config.SymBlock)

	mb.On("Subscribe", true).Return(2, chBlock)
	sim.Start()
	<-time.After(50 * time.Millisecond)

	chBlock <- config.SymBlock{
		BlockNumber: big.NewInt(5),
	}

	timer := time.NewTimer(time.Second)
	select {
	case <-sim.Notify():
		timer.Stop()
	case <-timer.C:
		assert.Fail(t, "no signal received to indicated new block")
	}

	block, err := sim.LatestBlockHeight(context.TODO())
	assert.NoError(t, err)
	assert.Equal(t, uint64(5), block)

	mb.On("Unsubscribe", 2).Return()
	sim.Stop()
}

type MockBlockBroadcaster struct {
	mock.Mock
}

func (_m *MockBlockBroadcaster) Subscribe(delay bool) (int, chan config.SymBlock) {
	ret := _m.Mock.Called(delay)

	var r0 int
	if rf, ok := ret.Get(0).(func() int); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(int)
		}
	}

	var r1 chan config.SymBlock
	if rf, ok := ret.Get(1).(func() chan config.SymBlock); ok {
		r1 = rf()
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(chan config.SymBlock)
		}
	}

	return r0, r1
}

func (_m *MockBlockBroadcaster) Unsubscribe(id int) {
	_m.Mock.Called(id)
}

func (_m *MockBlockBroadcaster) Transmit(data []byte, epoch uint32) error {
	return _m.Mock.Called(data, epoch).Error(0)
}

type MockDigester struct {
	mock.Mock
}

func (_m *MockDigester) ConfigDigest(config types.ContractConfig) (types.ConfigDigest, error) {
	ret := _m.Mock.Called(config)

	var r0 types.ConfigDigest
	if rf, ok := ret.Get(0).(func() types.ConfigDigest); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(types.ConfigDigest)
		}
	}

	return r0, ret.Error(1)
}

type MockEncoder struct {
	mock.Mock
}

func (_m *MockEncoder) EncodeReport(r []ktypes.UpkeepResult) ([]byte, error) {
	ret := _m.Mock.Called(r)

	var r0 []byte
	if rf, ok := ret.Get(0).(func() []byte); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]byte)
		}
	}

	return r0, ret.Error(1)
}

func (_m *MockEncoder) DecodeReport(b []byte) ([]ktypes.UpkeepResult, error) {
	ret := _m.Mock.Called(b)

	var r0 []ktypes.UpkeepResult
	if rf, ok := ret.Get(0).(func() []ktypes.UpkeepResult); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]ktypes.UpkeepResult)
		}
	}

	return r0, ret.Error(1)
}
