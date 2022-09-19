package ocr2keepers

import (
	"testing"

	"github.com/smartcontractkit/libocr/commontypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestLogWriter(t *testing.T) {
	m := new(MockLogger)
	lw := &logWriter{l: m}
	input := []byte("test")

	m.On("Debug", string(input), mock.Anything)

	n, err := lw.Write(input)
	assert.NoError(t, err)
	assert.Equal(t, len(input), n)
}

var _ commontypes.Logger = (*MockLogger)(nil)

type MockLogger struct {
	mock.Mock
}

func (_m *MockLogger) Critical(msg string, fields commontypes.LogFields) {
	_m.Mock.Called(msg, fields)
}

func (_m *MockLogger) Error(msg string, fields commontypes.LogFields) {
	_m.Mock.Called(msg, fields)
}

func (_m *MockLogger) Warn(msg string, fields commontypes.LogFields) {
	_m.Mock.Called(msg, fields)
}

func (_m *MockLogger) Info(msg string, fields commontypes.LogFields) {
	_m.Mock.Called(msg, fields)
}

func (_m *MockLogger) Debug(msg string, fields commontypes.LogFields) {
	_m.Mock.Called(msg, fields)
}

func (_m *MockLogger) Trace(msg string, fields commontypes.LogFields) {
	_m.Mock.Called(msg, fields)
}
