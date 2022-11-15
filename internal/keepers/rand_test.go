package keepers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCryptoRandSource(t *testing.T) {
	s := newCryptoRandSource()
	i := s.Int63()
	assert.NotEqual(t, 0, i)
}
