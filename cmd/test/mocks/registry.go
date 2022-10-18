package mocks

import (
	"context"

	"github.com/smartcontractkit/ocr2keepers/pkg/types"
	"github.com/stretchr/testify/mock"
)

type MockRegistry struct {
	mock.Mock
}

func (_m *MockRegistry) GetActiveUpkeepKeys(ctx context.Context, key types.BlockKey) ([]types.UpkeepKey, error) {
	ret := _m.Mock.Called(ctx, key)

	var r0 []types.UpkeepKey
	if rf, ok := ret.Get(0).(func() []types.UpkeepKey); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]types.UpkeepKey)
		}
	}

	return r0, ret.Error(1)
}

func (_m *MockRegistry) CheckUpkeep(ctx context.Context, key types.UpkeepKey) (bool, types.UpkeepResult, error) {
	ret := _m.Mock.Called(ctx, key)

	var r1 types.UpkeepResult
	if rf, ok := ret.Get(1).(func() types.UpkeepResult); ok {
		r1 = rf()
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(types.UpkeepResult)
		}
	}

	return ret.Bool(0), r1, ret.Error(2)
}

func (_m *MockRegistry) IdentifierFromKey(key types.UpkeepKey) (types.UpkeepIdentifier, error) {
	ret := _m.Mock.Called(key)

	var r0 types.UpkeepIdentifier
	if rf, ok := ret.Get(0).(func() types.UpkeepIdentifier); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(types.UpkeepIdentifier)
		}
	}

	return r0, ret.Error(1)
}
