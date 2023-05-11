package ocr2keepers

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestObservation_UnmarshalJSON(t *testing.T) {
	t.Run("valid bytes unmarshal successfully", func(t *testing.T) {
		var observation Observation
		err := json.Unmarshal([]byte(`{"1":"123","2":["NDU2"]}`), &observation)
		assert.Nil(t, err)
		assert.Equal(t, Observation{
			BlockKey: BlockKey("123"),
			UpkeepIdentifiers: []UpkeepIdentifier{
				UpkeepIdentifier("456"),
			},
		}, observation)
	})

	t.Run("invalid bytes unmarshal unsuccessfully", func(t *testing.T) {
		var observation Observation
		err := json.Unmarshal([]byte(`{"1":"123","2":"NDU2"}`), &observation)
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
