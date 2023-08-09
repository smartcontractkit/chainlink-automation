package ocr2keepers_test

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

	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg"
	"github.com/smartcontractkit/ocr2keepers/pkg/v2/encoding"
)

func TestObservation_UnmarshalJSON(t *testing.T) {
	t.Run("valid bytes unmarshal successfully", func(t *testing.T) {
		var observation ocr2keepers.Observation
		err := json.Unmarshal([]byte(`{"1":"123","2":["NDU2"]}`), &observation)
		assert.Nil(t, err)
		assert.Equal(t, ocr2keepers.Observation{
			BlockKey: ocr2keepers.BlockKey("123"),
			UpkeepIdentifiers: []ocr2keepers.UpkeepIdentifier{
				ocr2keepers.UpkeepIdentifier("456"),
			},
		}, observation)
	})

	t.Run("invalid bytes unmarshal unsuccessfully", func(t *testing.T) {
		var observation ocr2keepers.Observation
		err := json.Unmarshal([]byte(`{"1":"123","2":"NDU2"}`), &observation)
		assert.NotNil(t, err)
	})
}

func TestObservation_Validate(t *testing.T) {
	t.Run("returns error on block key validation error", func(t *testing.T) {
		var observation ocr2keepers.Observation

		errExpected := fmt.Errorf("expected error")
		mv := new(MockObservationValidator)

		mv.On("ValidateBlockKey", mock.Anything).Return(false, errExpected)

		err := observation.Validate(mv)

		assert.ErrorIs(t, err, errExpected)

		mv.AssertExpectations(t)
	})

	t.Run("returns error on block key invalid without error", func(t *testing.T) {
		var observation ocr2keepers.Observation

		mv := new(MockObservationValidator)

		mv.On("ValidateBlockKey", mock.Anything).Return(false, nil)

		err := observation.Validate(mv)

		assert.ErrorIs(t, err, ocr2keepers.ErrInvalidBlockKey)

		mv.AssertExpectations(t)
	})

	t.Run("does not return error on no upkeep identifiers in list", func(t *testing.T) {
		var observation ocr2keepers.Observation

		mv := new(MockObservationValidator)

		mv.On("ValidateBlockKey", mock.Anything).Return(true, nil)
		// no calls to ValidateUpkeepIdentifier because none in list

		err := observation.Validate(mv)

		assert.NoError(t, err)

		mv.AssertExpectations(t)
	})

	t.Run("returns error on 1st upkeep identifier validation error", func(t *testing.T) {
		observation := ocr2keepers.Observation{
			UpkeepIdentifiers: []ocr2keepers.UpkeepIdentifier{
				ocr2keepers.UpkeepIdentifier([]byte("123")),
				ocr2keepers.UpkeepIdentifier([]byte("345")),
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
		observation := ocr2keepers.Observation{
			UpkeepIdentifiers: []ocr2keepers.UpkeepIdentifier{
				ocr2keepers.UpkeepIdentifier([]byte("123")),
				ocr2keepers.UpkeepIdentifier([]byte("345")),
			},
		}

		mv := new(MockObservationValidator)

		mv.On("ValidateBlockKey", mock.Anything).Return(true, nil)
		mv.On("ValidateUpkeepIdentifier", mock.Anything).Return(true, nil).Twice()

		err := observation.Validate(mv)

		assert.NoError(t, err)

		mv.AssertExpectations(t)
	})

	t.Run("an ErrInvalidUpkeepIdentifier error is received", func(t *testing.T) {
		observation := ocr2keepers.Observation{
			UpkeepIdentifiers: []ocr2keepers.UpkeepIdentifier{
				ocr2keepers.UpkeepIdentifier([]byte("123")),
				ocr2keepers.UpkeepIdentifier([]byte("345")),
			},
		}

		err := observation.Validate(&mockValidator{
			ValidateBlockKeyFn: func(key ocr2keepers.BlockKey) (bool, error) {
				return true, nil
			},
			ValidateUpkeepIdentifierFn: func(identifier ocr2keepers.UpkeepIdentifier) (bool, error) {
				return false, nil
			},
		})

		assert.ErrorIs(t, err, ocr2keepers.ErrInvalidUpkeepIdentifier)
	})
}

type mockValidator struct {
	ValidateUpkeepKeyFn        func(ocr2keepers.UpkeepKey) (bool, error)
	ValidateUpkeepIdentifierFn func(ocr2keepers.UpkeepIdentifier) (bool, error)
	ValidateBlockKeyFn         func(ocr2keepers.BlockKey) (bool, error)
}

func (m *mockValidator) ValidateUpkeepKey(u ocr2keepers.UpkeepKey) (bool, error) {
	return m.ValidateUpkeepKeyFn(u)
}

func (m *mockValidator) ValidateUpkeepIdentifier(u ocr2keepers.UpkeepIdentifier) (bool, error) {
	return m.ValidateUpkeepIdentifierFn(u)
}

func (m *mockValidator) ValidateBlockKey(b ocr2keepers.BlockKey) (bool, error) {
	return m.ValidateBlockKeyFn(b)
}

func TestObservationsToUpkeepKeys(t *testing.T) {
	obs := []ocr2keepers.Observation{
		{BlockKey: ocr2keepers.BlockKey("123"), UpkeepIdentifiers: []ocr2keepers.UpkeepIdentifier{ocr2keepers.UpkeepIdentifier("1234")}},
		{BlockKey: ocr2keepers.BlockKey("124"), UpkeepIdentifiers: []ocr2keepers.UpkeepIdentifier{ocr2keepers.UpkeepIdentifier("1234")}},
		{BlockKey: ocr2keepers.BlockKey("125"), UpkeepIdentifiers: []ocr2keepers.UpkeepIdentifier{ocr2keepers.UpkeepIdentifier("1234")}},
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
	mc.On("GetMedian", mock.Anything).Return(ocr2keepers.BlockKey("124")).Once()
	mb.On("MakeUpkeepKey", mock.AnythingOfType("BlockKey"), mock.AnythingOfType("UpkeepIdentifier")).Return(ocr2keepers.UpkeepKey("124|1234")).Times(3)

	keys, err := ocr2keepers.ObservationsToUpkeepKeys(
		attr,
		mv,
		mc,
		mb,
		logger,
	)

	expected := [][]ocr2keepers.UpkeepKey{
		{ocr2keepers.UpkeepKey("124|1234")},
		{ocr2keepers.UpkeepKey("124|1234")},
		{ocr2keepers.UpkeepKey("124|1234")},
	}

	assert.NoError(t, err)
	assert.Equal(t, expected, keys)
}

type mockMedianCalculator struct {
	GetMedianFn func([]ocr2keepers.BlockKey) ocr2keepers.BlockKey
}

func (c *mockMedianCalculator) GetMedian(b []ocr2keepers.BlockKey) ocr2keepers.BlockKey {
	return c.GetMedianFn(b)
}

func TestObservationsToUpkeepKeys_boundsCheck(t *testing.T) {
	obs := []ocr2keepers.Observation{
		{BlockKey: ocr2keepers.BlockKey("18446744073709551616"), UpkeepIdentifiers: []ocr2keepers.UpkeepIdentifier{ocr2keepers.UpkeepIdentifier("2"), ocr2keepers.UpkeepIdentifier("3")}},
		{BlockKey: ocr2keepers.BlockKey("1"), UpkeepIdentifiers: []ocr2keepers.UpkeepIdentifier{ocr2keepers.UpkeepIdentifier("115792089237316195423570985008687907853269984665640564039457584007913129639936"), ocr2keepers.UpkeepIdentifier("3")}},
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

	validator := encoding.BasicEncoder{}

	medianCalculator := &mockMedianCalculator{
		GetMedianFn: func(keys []ocr2keepers.BlockKey) ocr2keepers.BlockKey {
			return keys[0]
		},
	}

	logger := log.New(io.Discard, "", 0)

	_, err := ocr2keepers.ObservationsToUpkeepKeys(attr, validator, medianCalculator, nil, logger)
	assert.ErrorContains(t, err, "observations not properly encoded")

}
func TestObservationsToUpkeepKeys_Empty(t *testing.T) {
	obs := []ocr2keepers.Observation{
		{BlockKey: ocr2keepers.BlockKey("123"), UpkeepIdentifiers: []ocr2keepers.UpkeepIdentifier{}},
		{BlockKey: ocr2keepers.BlockKey("124"), UpkeepIdentifiers: []ocr2keepers.UpkeepIdentifier{}},
		{BlockKey: ocr2keepers.BlockKey("125"), UpkeepIdentifiers: []ocr2keepers.UpkeepIdentifier{}},
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
	mc.On("GetMedian", mock.Anything).Return(ocr2keepers.BlockKey("124")).Once()

	keys, err := ocr2keepers.ObservationsToUpkeepKeys(
		attr,
		mv,
		mc,
		nil,
		logger,
	)

	expected := [][]ocr2keepers.UpkeepKey{}

	assert.NoError(t, err)
	assert.Equal(t, expected, keys)
}

type MockObservationValidator struct {
	mock.Mock
}

func (_m *MockObservationValidator) ValidateUpkeepKey(key ocr2keepers.UpkeepKey) (bool, error) {
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

func (_m *MockObservationValidator) ValidateUpkeepIdentifier(id ocr2keepers.UpkeepIdentifier) (bool, error) {
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

func (_m *MockObservationValidator) ValidateBlockKey(key ocr2keepers.BlockKey) (bool, error) {
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

func (_m *MockMedianCalculator) GetMedian(keys []ocr2keepers.BlockKey) ocr2keepers.BlockKey {
	ret := _m.Called(keys)

	var r0 ocr2keepers.BlockKey
	if rf, ok := ret.Get(0).(func() ocr2keepers.BlockKey); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(ocr2keepers.BlockKey)
		}
	}

	return r0
}

type MockBuilder struct {
	mock.Mock
}

func (_m *MockBuilder) MakeUpkeepKey(key ocr2keepers.BlockKey, id ocr2keepers.UpkeepIdentifier) ocr2keepers.UpkeepKey {
	ret := _m.Called(key, id)

	var r0 ocr2keepers.UpkeepKey
	if rf, ok := ret.Get(0).(func() ocr2keepers.UpkeepKey); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(ocr2keepers.UpkeepKey)
		}
	}

	return r0
}
