package ocr2keepers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"testing"

	"github.com/smartcontractkit/libocr/commontypes"
	"github.com/smartcontractkit/libocr/offchainreporting2plus/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestObservation_UnmarshalJSON(t *testing.T) {
	t.Run("valid bytes unmarshal successfully", func(t *testing.T) {
		var observation Observation
		err := json.Unmarshal([]byte(`{"1":{"block": 123},"2":["NDU2"]}`), &observation)
		assert.Nil(t, err)
		assert.Equal(t, Observation{
			BlockKey: BlockKey{Block: 123},
			UpkeepIdentifiers: []UpkeepIdentifier{
				UpkeepIdentifier("456"),
			},
		}, observation)
	})

	t.Run("invalid bytes unmarshal unsuccessfully", func(t *testing.T) {
		var observation Observation
		err := json.Unmarshal([]byte(`{"1":{"block":123},"2":"NDU2"}`), &observation)
		assert.NotNil(t, err)
	})
}

func TestObservation_Validate(t *testing.T) {
	t.Run("returns error on block key validation error", func(t *testing.T) {
		var observation Observation

		errExpected := fmt.Errorf("expected error")
		mv := new(MockObservationValidator)

		mv.On("ValidateBlockKey", mock.Anything).Return(false, errExpected)

		err := observation.Validate(mv)

		assert.ErrorIs(t, err, errExpected)

		mv.AssertExpectations(t)
	})

	t.Run("returns error on block key invalid without error", func(t *testing.T) {
		var observation Observation

		mv := new(MockObservationValidator)

		mv.On("ValidateBlockKey", mock.Anything).Return(false, nil)

		err := observation.Validate(mv)

		assert.ErrorIs(t, err, ErrInvalidBlockKey)

		mv.AssertExpectations(t)
	})

	t.Run("does not return error on no upkeep identifiers in list", func(t *testing.T) {
		var observation Observation

		mv := new(MockObservationValidator)

		mv.On("ValidateBlockKey", mock.Anything).Return(true, nil)
		// no calls to ValidateUpkeepIdentifier because none in list

		err := observation.Validate(mv)

		assert.NoError(t, err)

		mv.AssertExpectations(t)
	})

	t.Run("returns error on 1st upkeep identifier validation error", func(t *testing.T) {
		observation := Observation{
			UpkeepIdentifiers: []UpkeepIdentifier{
				UpkeepIdentifier([]byte("123")),
				UpkeepIdentifier([]byte("345")),
			},
		}

		mv := new(MockObservationValidator)

		errExpected := fmt.Errorf("expected error")

		mv.On("ValidateBlockKey", mock.Anything).Return(true, nil)
		mv.On("ValidateUpkeepIdentifier", mock.Anything).Return(false, errExpected).Once()

		err := observation.Validate(mv)

		assert.ErrorIs(t, err, errExpected)

		mv.AssertExpectations(t)
	})

	t.Run("returns no validation error", func(t *testing.T) {
		observation := Observation{
			UpkeepIdentifiers: []UpkeepIdentifier{
				UpkeepIdentifier([]byte("123")),
				UpkeepIdentifier([]byte("345")),
			},
		}

		mv := new(MockObservationValidator)

		mv.On("ValidateBlockKey", mock.Anything).Return(true, nil)
		mv.On("ValidateUpkeepIdentifier", mock.Anything).Return(true, nil).Twice()

		err := observation.Validate(mv)

		assert.NoError(t, err)

		mv.AssertExpectations(t)
	})
}

func TestObservationsToUpkeepKeys(t *testing.T) {
	obs := []Observation{
		{BlockKey: BlockKey{Block: 123}, UpkeepIdentifiers: []UpkeepIdentifier{UpkeepIdentifier("1234")}},
		{BlockKey: BlockKey{Block: 124}, UpkeepIdentifiers: []UpkeepIdentifier{UpkeepIdentifier("1234")}},
		{BlockKey: BlockKey{Block: 125}, UpkeepIdentifiers: []UpkeepIdentifier{UpkeepIdentifier("1234")}},
	}

	attr := make([]types.AttributedObservation, len(obs))
	for i, o := range obs {
		b, err := json.Marshal(o)
		if err != nil {
			t.Logf("json marshal error: %s", err)
			t.FailNow()
		}

		attr[i] = types.AttributedObservation{
			Observer:    commontypes.OracleID(i),
			Observation: types.Observation(b),
		}
	}

	logger := log.New(io.Discard, "", 0)
	mv := new(MockObservationValidator)
	mc := new(MockMedianCalculator)
	mb := new(MockBuilder)

	mv.On("ValidateBlockKey", mock.AnythingOfType("BlockKey")).Return(true, nil).Times(3)
	mv.On("ValidateUpkeepIdentifier", mock.AnythingOfType("UpkeepIdentifier")).Return(true, nil).Times(3)
	mc.On("GetMedian", mock.Anything).Return(BlockKey{Block: 124}).Once()
	mb.On("MakeUpkeepKey", mock.AnythingOfType("BlockKey"), mock.AnythingOfType("UpkeepIdentifier")).Return(UpkeepKey("124|1234")).Times(3)

	keys, err := observationsToUpkeepKeys(
		attr,
		mv,
		mc,
		mb,
		logger,
	)

	expected := [][]UpkeepKey{
		{UpkeepKey("124|1234")},
		{UpkeepKey("124|1234")},
		{UpkeepKey("124|1234")},
	}

	assert.NoError(t, err)
	assert.Equal(t, expected, keys)
}

func TestObservationsToUpkeepKeys_Empty(t *testing.T) {
	obs := []Observation{
		{BlockKey: BlockKey{Block: 123}, UpkeepIdentifiers: []UpkeepIdentifier{}},
		{BlockKey: BlockKey{Block: 124}, UpkeepIdentifiers: []UpkeepIdentifier{}},
		{BlockKey: BlockKey{Block: 125}, UpkeepIdentifiers: []UpkeepIdentifier{}},
	}

	attr := make([]types.AttributedObservation, len(obs))
	for i, o := range obs {
		b, err := json.Marshal(o)
		if err != nil {
			t.Logf("json marshal error: %s", err)
			t.FailNow()
		}

		attr[i] = types.AttributedObservation{
			Observer:    commontypes.OracleID(i),
			Observation: types.Observation(b),
		}
	}

	logger := log.New(io.Discard, "", 0)
	mv := new(MockObservationValidator)
	mc := new(MockMedianCalculator)

	mv.On("ValidateBlockKey", mock.AnythingOfType("BlockKey")).Return(true, nil).Times(3)
	mc.On("GetMedian", mock.Anything).Return(BlockKey{Block: 124}).Once()

	keys, err := observationsToUpkeepKeys(
		attr,
		mv,
		mc,
		nil,
		logger,
	)

	expected := [][]UpkeepKey{}

	assert.NoError(t, err)
	assert.Equal(t, expected, keys)
}

type MockObservationValidator struct {
	mock.Mock
}

func (_m *MockObservationValidator) ValidateUpkeepKey(key UpkeepKey) (bool, error) {
	ret := _m.Called(key)

	var r0 bool
	if rf, ok := ret.Get(0).(func() bool); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(bool)
		}
	}

	return r0, ret.Error(1)
}

func (_m *MockObservationValidator) ValidateUpkeepIdentifier(id UpkeepIdentifier) (bool, error) {
	ret := _m.Called(id)

	var r0 bool
	if rf, ok := ret.Get(0).(func() bool); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(bool)
		}
	}

	return r0, ret.Error(1)
}

func (_m *MockObservationValidator) ValidateBlockKey(key BlockKey) (bool, error) {
	ret := _m.Called(key)

	var r0 bool
	if rf, ok := ret.Get(0).(func() bool); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(bool)
		}
	}

	return r0, ret.Error(1)
}

type MockMedianCalculator struct {
	mock.Mock
}

func (_m *MockMedianCalculator) GetMedian(keys []BlockKey) BlockKey {
	ret := _m.Called(keys)

	var r0 BlockKey
	if rf, ok := ret.Get(0).(func() BlockKey); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(BlockKey)
		}
	}

	return r0
}

type MockBuilder struct {
	mock.Mock
}

func (_m *MockBuilder) MakeUpkeepKey(key BlockKey, id UpkeepIdentifier) UpkeepKey {
	ret := _m.Called(key, id)

	var r0 UpkeepKey
	if rf, ok := ret.Get(0).(func() UpkeepKey); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(UpkeepKey)
		}
	}

	return r0
}
