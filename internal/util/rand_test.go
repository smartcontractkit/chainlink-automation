package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCryptoRandSource(t *testing.T) {
	s := NewCryptoRandSource()
	i := s.Int63()
	assert.NotEqual(t, 0, i)
}
