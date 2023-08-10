package ocr2keepers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestFilterAndDedupe(t *testing.T) {
	inputs := [][]UpkeepKey{
		{UpkeepKey("123|1234"), UpkeepKey("124|1234")},
		{UpkeepKey("123|1234")},
		{UpkeepKey("125|1234"), UpkeepKey("124|1235")},
	}

	expected := []UpkeepKey{
		UpkeepKey("123|1234"),
		UpkeepKey("124|1234"),
		UpkeepKey("125|1234"),
		UpkeepKey("124|1235"),
	}

	mf := new(MockFilter)

	mf.On("IsPending", mock.AnythingOfType("UpkeepKey")).Return(false, nil).Times(5)

	results, err := filterAndDedupe(inputs, mf.IsPending)

	assert.NoError(t, err)
	assert.Equal(t, expected, results)

	mf.AssertExpectations(t)
}

type MockFilter struct {
	mock.Mock
}

func (_m *MockFilter) IsPending(key UpkeepKey) (bool, error) {
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
