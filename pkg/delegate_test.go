package ocr2keepers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStart(t *testing.T) {
	d, _ := NewDelegate(DelegateConfig{})
	err := d.Start()
	assert.Equal(t, err.Error(), "unimplemented")
}

func TestClose(t *testing.T) {
	var err error

	d, err := NewDelegate(DelegateConfig{})
	assert.NoError(t, err)
	if err != nil {
		t.FailNow()
	}

	assert.NotNil(t, d.keeper, "Delegate keeper should not be nil")
	if d.keeper == nil {
		t.FailNow()
	}

	err = d.Close()
	assert.Equal(t, err.Error(), "can only close a started Oracle: stopping keeper oracle")
}
