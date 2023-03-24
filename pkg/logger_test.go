package ocr2keepers

import (
	"testing"

	"github.com/smartcontractkit/ocr2keepers/pkg/types/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestLogWriter(t *testing.T) {
	m := mocks.NewMockLogger(t)
	lw := &logWriter{l: m}
	input := []byte("test")

	m.On("Debug", string(input), mock.Anything)

	n, err := lw.Write(input)
	assert.NoError(t, err)
	assert.Equal(t, len(input), n)

	m.AssertExpectations(t)
}
