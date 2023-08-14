package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCloser(t *testing.T) {
	called := false
	cancelFn := func() {
		called = true
	}
	closer := &Closer{}

	ok := closer.Close()
	assert.False(t, ok)

	ok = closer.Store(cancelFn)
	assert.True(t, ok)

	ok = closer.Store(cancelFn)
	assert.False(t, ok)

	ok = closer.Close()
	assert.True(t, ok)

	assert.True(t, called)
}
