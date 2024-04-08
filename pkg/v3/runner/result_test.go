package runner

import (
	"fmt"
	ocr2keepers "github.com/smartcontractkit/chainlink-common/pkg/types/automation"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestResultAdder(t *testing.T) {
	t.Run("Successes", func(t *testing.T) {
		resultStruct := newResult()
		expected := 100

		for x := 0; x < expected; x++ {
			resultStruct.AddSuccesses(1)
		}

		assert.Equal(t, expected, resultStruct.Successes(), "success count should match expected")
	})

	t.Run("Failures", func(t *testing.T) {
		resultStruct := newResult()
		expected := 100

		for x := 0; x < expected; x++ {
			resultStruct.AddFailures(1)
		}

		assert.Equal(t, expected, resultStruct.Failures(), "failure count should match expected")
	})

	t.Run("Errors", func(t *testing.T) {
		resultStruct := newResult()
		expected := fmt.Errorf("expected error")

		resultStruct.SetErr(fmt.Errorf("initial error"))
		resultStruct.SetErr(expected)

		assert.ErrorIs(t, resultStruct.Err(), expected, "error returned should be last error set")
	})

	t.Run("Rates", func(t *testing.T) {
		resultStruct := newResult()

		for x := 1; x <= 100; x++ {
			resultStruct.AddSuccesses(1)

			if x%5 == 0 {
				resultStruct.AddFailures(1)
			}
		}

		const expectedTotal int = 120
		const expectedSuccess float64 = 100.0 / 120.0
		const expectedFailure float64 = 20.0 / 120.0

		assert.Equal(t, expectedTotal, resultStruct.Total(), "total should return the total number of sucesses and failures")
		assert.Equal(t, expectedSuccess, resultStruct.SuccessRate(), "success rate should return the number of successes divided by the total results")
		assert.Equal(t, expectedFailure, resultStruct.FailureRate(), "failure rate should return the number of failures divided by the total results")
	})

	t.Run("AddResults", func(t *testing.T) {
		resultStruct := newResult()

		expected := []ocr2keepers.CheckResult{}
		for x := 0; x <= 100; x++ {
			x := ocr2keepers.CheckResult{
				WorkID: fmt.Sprintf("%d", x),
			}
			resultStruct.Add(x)
			expected = append(expected, x)
		}

		assert.Equal(t, expected, resultStruct.Values(), "all values added should be returned")
	})
}

func TestConcurrentResult(t *testing.T) {
	var wg sync.WaitGroup

	resultStruct := newResult()

	// add successes and failures in one thread
	wg.Add(1)
	go func(r *result) {
		<-time.After(time.Second)
		for x := 1; x <= 10000; x++ {
			resultStruct.AddSuccesses(1)

			if x%5 == 0 {
				resultStruct.AddFailures(1)
			}
		}
		wg.Done()
	}(resultStruct)

	// add values in another
	wg.Add(1)
	go func(r *result) {
		<-time.After(time.Second)
		for x := 1; x <= 12000; x++ {
			resultStruct.Add(ocr2keepers.CheckResult{
				WorkID: fmt.Sprintf("%d", x),
			})
		}
		wg.Done()
	}(resultStruct)

	// repeatedly ask for the stats in another
	wg.Add(1)
	go func(r *result) {
		<-time.After(time.Second)
		for x := 0; x <= 12000; x++ {
			_ = resultStruct.Failures()
			_ = resultStruct.FailureRate()
			_ = resultStruct.Successes()
			_ = resultStruct.SuccessRate()
			_ = resultStruct.Total()
		}
		wg.Done()
	}(resultStruct)

	wg.Wait()

	// assert values are correct with no panics
	assert.Equal(t, 10000, resultStruct.Successes(), "success count should match expected")
	assert.Equal(t, 2000, resultStruct.Failures(), "failure count should match expected")
	assert.Equal(t, 12000, resultStruct.Total(), "total count should match expected")
	assert.Len(t, resultStruct.Values(), resultStruct.Total(), "total values '%d' should match total count '%d'", len(resultStruct.Values()), resultStruct.Total())
}
