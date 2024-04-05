package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResults_SuccessRate(t *testing.T) {
	result := &Results{
		Failures:  10,
		Successes: 90,
	}

	assert.Equal(t, result.SuccessRate(), .9)

	result = &Results{
		Successes: 0,
	}

	assert.Equal(t, result.SuccessRate(), float64(0))
}

func TestResults_FailureRate(t *testing.T) {
	result := &Results{
		Failures:  10,
		Successes: 90,
	}

	assert.Equal(t, result.FailureRate(), .1)

	result = &Results{
		Failures: 0,
	}

	assert.Equal(t, result.FailureRate(), float64(0))
}
