package mocks

import "github.com/stretchr/testify/mock"

type MockMonitoringEndpoint struct {
	mock.Mock
}

func (_m *MockMonitoringEndpoint) SendLog(log []byte) {
	_m.Mock.Called(log)
}
