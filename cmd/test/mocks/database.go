package mocks

import (
	"context"
	"time"

	"github.com/smartcontractkit/libocr/offchainreporting/types"
	"github.com/stretchr/testify/mock"
)

type MockDatabase struct {
	mock.Mock
}

func (_m *MockDatabase) ReadState(ctx context.Context, configDigest types.ConfigDigest) (*types.PersistentState, error) {
}
func (_m *MockDatabase) WriteState(ctx context.Context, configDigest types.ConfigDigest, state types.PersistentState) error {
}

func (_m *MockDatabase) ReadConfig(ctx context.Context) (*types.ContractConfig, error)      {}
func (_m *MockDatabase) WriteConfig(ctx context.Context, config types.ContractConfig) error {}

func (_m *MockDatabase) StorePendingTransmission(context.Context, types.PendingTransmissionKey, types.PendingTransmission) error {
}
func (_m *MockDatabase) PendingTransmissionsWithConfigDigest(context.Context, types.ConfigDigest) (map[types.PendingTransmissionKey]types.PendingTransmission, error) {
}
func (_m *MockDatabase) DeletePendingTransmission(context.Context, types.PendingTransmissionKey) error {
}
func (_m *MockDatabase) DeletePendingTransmissionsOlderThan(context.Context, time.Time) error {}
