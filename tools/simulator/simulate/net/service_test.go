package net_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/smartcontractkit/chainlink-automation/tools/simulator/simulate/net"
)

func TestSimulatedNetworkService(t *testing.T) {
	load := 0.999
	rate := 1
	latency := 100
	netTelemetry := new(mockNetTelemetry)

	service := net.NewSimulatedNetworkService(load, rate, latency, netTelemetry)

	netTelemetry.On("Register", "test", mock.Anything, mock.Anything)
	netTelemetry.On("AddRateDataPoint", mock.Anything).Maybe()

	err := <-service.Call(context.Background(), "test")

	assert.NotNil(t, err)
}

type mockNetTelemetry struct {
	mock.Mock
}

func (_m *mockNetTelemetry) Register(name string, duration time.Duration, err error) {
	_m.Called(name, duration, err)
}

func (_m *mockNetTelemetry) AddRateDataPoint(quantity int) {
	_m.Called(quantity)
}
