package tickers

import (
	"context"
	"fmt"
	"io"
	"log"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg/v3/types"
)

type mockUpkeepObserver struct {
	processFn func(context.Context, Tick[[]ocr2keepers.UpkeepPayload]) error
}

func (o *mockUpkeepObserver) Process(ctx context.Context, t Tick[[]ocr2keepers.UpkeepPayload]) error {
	return o.processFn(ctx, t)
}

func TestSampleTicker(t *testing.T) {
	tests := []struct {
		Name                 string
		TestData             []ocr2keepers.UpkeepPayload
		ExpectedSampleCount  int
		ExpectedSampleResult int
	}{
		{
			Name: "simple happy path",
			TestData: []ocr2keepers.UpkeepPayload{
				{ID: "1"},
			},
			ExpectedSampleCount:  1,
			ExpectedSampleResult: 1,
		},
		{
			Name: "reduce to sample size",
			TestData: []ocr2keepers.UpkeepPayload{
				{ID: "1"},
				{ID: "2"},
				{ID: "3"},
			},
			ExpectedSampleCount:  2,
			ExpectedSampleResult: 2,
		},
		{
			Name:                 "empty set",
			TestData:             []ocr2keepers.UpkeepPayload{},
			ExpectedSampleCount:  2,
			ExpectedSampleResult: 0,
		},
	}

	// setup common ticker for all tests
	var (
		wg        sync.WaitGroup
		mu        sync.Mutex
		processed int
	)

	// create basic mocks
	mg := new(MockGetter)
	mr := new(MockRatio)

	// create an observer that tracks sampling and mocks pipeline results
	mockObserver := &mockUpkeepObserver{
		processFn: func(ctx context.Context, t Tick[[]ocr2keepers.UpkeepPayload]) error {
			upkeeps, _ := t.Value(ctx)

			// assert that retry was in tick at least once
			mu.Lock()
			processed = len(upkeeps)
			mu.Unlock()

			return nil
		},
	}

	// create mocked block source
	ch := make(chan ocr2keepers.BlockHistory)
	sub := &mockSubscriber{
		SubscribeFn: func() (int, chan ocr2keepers.BlockHistory, error) {
			return 0, ch, nil
		},
		UnsubscribeFn: func(id int) error {
			return nil
		},
	}

	// Create a sampleTicker instance
	rt, err := NewSampleTicker(
		mr,
		mg,
		mockObserver,
		sub,
		log.New(io.Discard, "", 0),
	)
	assert.NoError(t, err, "no error on instantiation")

	// start the ticker in a separate thread
	wg.Add(1)
	go func() {
		assert.NoError(t, rt.Start(context.Background()))
		wg.Done()
	}()

	// run all tests on the same ticker instance
	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			mg.On("GetActiveUpkeeps", mock.Anything, mock.Anything).Return(test.TestData, nil)
			mr.On("OfInt", mock.Anything).Return(test.ExpectedSampleCount)

			// send a block history
			ch <- []ocr2keepers.BlockKey{
				{
					Number: 4,
				},
				{
					Number: 3,
				},
				{
					Number: 2,
				},
			}

			// wait a little longer than the sampling timeout
			time.Sleep(20 * time.Millisecond)

			// test expectations
			mg.AssertExpectations(t)
			mr.AssertExpectations(t)

			// reset for next test
			mg.ExpectedCalls = []*mock.Call{}
			mr.ExpectedCalls = []*mock.Call{}

			mu.Lock()
			assert.Equal(t, test.ExpectedSampleResult, processed, "tick should have been sampled exactly %d times", test.ExpectedSampleResult)
			mu.Unlock()
		})
	}

	assert.NoError(t, rt.Close(), "no error on close")

	wg.Wait()
}

func TestSampleTicker_ErrorStates(t *testing.T) {
	tests := []struct {
		Name                 string
		TestData             []ocr2keepers.UpkeepPayload
		ExpectedSampleCount  int
		ExpectedSampleResult int
		GetterFnError        error
		ObserverFnError      error
		ExpectedErr          string
	}{
		{
			Name:                 "getter function error",
			TestData:             []ocr2keepers.UpkeepPayload{},
			ExpectedSampleCount:  1,
			ExpectedSampleResult: 0,
			GetterFnError:        fmt.Errorf("test"),
			ObserverFnError:      nil,
			ExpectedErr:          "failed to get upkeeps",
		},
		{
			Name:                 "observer function error",
			TestData:             []ocr2keepers.UpkeepPayload{},
			ExpectedSampleCount:  1,
			ExpectedSampleResult: 0,
			GetterFnError:        nil,
			ObserverFnError:      fmt.Errorf("test"),
			ExpectedErr:          "error processing observer",
		},
	}

	// run all tests on a different ticker instance
	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			// setup common ticker for all tests
			var (
				wg        sync.WaitGroup
				mu        sync.Mutex
				processed int
			)

			// create basic mocks
			mg := new(MockGetter)
			mr := new(MockRatio)

			// create an observer that tracks sampling and mocks pipeline results
			mockObserver := &mockUpkeepObserver{
				processFn: func(ctx context.Context, t Tick[[]ocr2keepers.UpkeepPayload]) error {
					upkeeps, _ := t.Value(ctx)

					// assert that retry was in tick at least once
					mu.Lock()
					processed = len(upkeeps)
					mu.Unlock()

					return test.ObserverFnError
				},
			}

			// create mocked block source
			ch := make(chan ocr2keepers.BlockHistory)
			sub := &mockSubscriber{
				SubscribeFn: func() (int, chan ocr2keepers.BlockHistory, error) {
					return 0, ch, nil
				},
				UnsubscribeFn: func(id int) error {
					return nil
				},
			}

			b := new(strings.Builder)

			// Create a sampleTicker instance
			rt, err := NewSampleTicker(
				mr,
				mg,
				mockObserver,
				sub,
				log.New(b, "[test-log] ", log.LstdFlags),
			)
			assert.NoError(t, err, "no error on instantiation")

			// start the ticker in a separate thread
			wg.Add(1)
			go func() {
				assert.NoError(t, rt.Start(context.Background()))
				wg.Done()
			}()

			mg.On("GetActiveUpkeeps", mock.Anything, mock.Anything).Return(test.TestData, test.GetterFnError)

			if test.GetterFnError == nil {
				mr.On("OfInt", mock.Anything).Return(test.ExpectedSampleCount)
			}

			// send a block history
			ch <- []ocr2keepers.BlockKey{
				{
					Number: 4,
				},
				{
					Number: 3,
				},
				{
					Number: 2,
				},
			}

			// wait a little longer than the sampling timeout
			time.Sleep(20 * time.Millisecond)

			// test expectations
			mg.AssertExpectations(t)
			mr.AssertExpectations(t)

			// reset for next test
			mg.ExpectedCalls = []*mock.Call{}
			mr.ExpectedCalls = []*mock.Call{}

			mu.Lock()
			assert.Equal(t, test.ExpectedSampleResult, processed, "tick should have been sampled exactly %d times", test.ExpectedSampleResult)
			mu.Unlock()

			assert.NoError(t, rt.Close(), "no error on close")
			wg.Wait()

			assert.Contains(t, b.String(), test.ExpectedErr, "should contain expected log: %s", test.ExpectedErr)
		})
	}
}

type MockRatio struct {
	mock.Mock
}

func (_m *MockRatio) OfInt(v int) int {
	ret := _m.Called(v)

	var r0 int
	if rf, ok := ret.Get(0).(func(int) int); ok {
		r0 = rf(v)
	} else {
		r0 = ret.Get(0).(int)
	}

	return r0
}

type MockGetter struct {
	mock.Mock
}

func (_m *MockGetter) GetActiveUpkeeps(ctx context.Context, bk ocr2keepers.BlockKey) ([]ocr2keepers.UpkeepPayload, error) {
	ret := _m.Called(ctx, bk)

	var r0 []ocr2keepers.UpkeepPayload
	var r1 error

	if rf, ok := ret.Get(0).(func(context.Context, ocr2keepers.BlockKey) ([]ocr2keepers.UpkeepPayload, error)); ok {
		return rf(ctx, bk)
	}

	if rf, ok := ret.Get(0).(func(context.Context, ocr2keepers.BlockKey) []ocr2keepers.UpkeepPayload); ok {
		r0 = rf(ctx, bk)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]ocr2keepers.UpkeepPayload)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, ocr2keepers.BlockKey) error); ok {
		r1 = rf(ctx, bk)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
