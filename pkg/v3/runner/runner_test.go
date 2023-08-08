package runner

import (
	"context"
	"fmt"
	"io"
	"log"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/smartcontractkit/ocr2keepers/pkg/v3/runner/mocks"
	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg/v3/types"
)

func TestRunnerCache(t *testing.T) {
	logger := log.New(io.Discard, "", 0)
	mr := new(mocks.MockRunnable)

	conf := RunnerConfig{
		Workers:           2,
		WorkerQueueLength: 1000,
		CacheExpire:       500 * time.Millisecond,
		CacheClean:        1 * time.Second,
	}

	runner, err := NewRunner(logger, mr, conf)
	assert.NoError(t, err, "no error should be encountered during runner creation")

	payloads := []ocr2keepers.UpkeepPayload{
		{WorkID: "a"},
		{WorkID: "b"},
		{WorkID: "c"},
		{WorkID: "d"},
		{WorkID: "e"},
	}

	expected := make([]ocr2keepers.CheckResult, len(payloads))
	for i := range payloads {
		expected[i] = ocr2keepers.CheckResult{
			Payload: payloads[i],
		}
	}

	// ensure that context and payloads are passed through to the runnable
	// return results that should be cached
	mr.On("CheckUpkeeps", append([]interface{}{mock.Anything}, toInterfaces(payloads...)...)...).Return(expected, nil).Once().After(500 * time.Millisecond)

	results, err := runner.CheckUpkeeps(context.Background(), payloads...)
	assert.NoError(t, err, "no error should be encountered during upkeep checking")
	assert.Equal(t, expected, results, "results should be returned without changes from the runnable")

	// ensure that a call with the same payloads uses cache instead of calling runnable
	results, err = runner.CheckUpkeeps(context.Background(), payloads...)
	assert.NoError(t, err, "no error should be encountered during upkeep checking")
	assert.Equal(t, expected, results, "results should be returned without changes from the runnable")
}

func TestRunnerBatching(t *testing.T) {
	logger := log.New(io.Discard, "", 0)
	mr := new(mocks.MockRunnable)

	conf := RunnerConfig{
		Workers:           2,
		WorkerQueueLength: 1000,
		CacheExpire:       500 * time.Millisecond,
		CacheClean:        1 * time.Second,
	}

	runner, err := NewRunner(logger, mr, conf)
	assert.NoError(t, err, "no error should be encountered during runner creation")

	payloads := []ocr2keepers.UpkeepPayload{
		{WorkID: "a"},
		{WorkID: "b"},
		{WorkID: "c"},
		{WorkID: "d"},
		{WorkID: "e"},
		{WorkID: "f"},
		{WorkID: "g"},
		{WorkID: "h"},
		{WorkID: "i"},
		{WorkID: "j"},
		{WorkID: "k"},
		{WorkID: "l"},
	}

	expected := make([]ocr2keepers.CheckResult, len(payloads))
	for i := range payloads {
		expected[i] = ocr2keepers.CheckResult{
			Payload: payloads[i],
		}
	}

	// ensure that context and payloads are passed through to the runnable
	// payloads and results should be split by batches
	mr.On("CheckUpkeeps", append([]interface{}{mock.Anything}, toInterfaces(payloads[:10]...)...)...).Return(expected[:10], nil).Once().After(500 * time.Millisecond)
	mr.On("CheckUpkeeps", append([]interface{}{mock.Anything}, toInterfaces(payloads[10:]...)...)...).Return(expected[10:], nil).Once().After(500 * time.Millisecond)

	// all batches should be collected into a single result set
	results, err := runner.CheckUpkeeps(context.Background(), payloads...)

	// sort the results for comparison
	sort.Slice(results, func(i, j int) bool {
		return results[i].Payload.WorkID < results[j].Payload.WorkID
	})

	assert.NoError(t, err, "no error should be encountered during upkeep checking")
	assert.Equal(t, expected, results, "results should be returned without changes from the runnable")
}

func TestRunnerConcurrent(t *testing.T) {
	// test that multiple calls to the runner are run concurrently and the results
	// are return separately

	logger := log.New(io.Discard, "", 0)
	mr := new(mocks.MockRunnable)

	conf := RunnerConfig{
		Workers:           2,
		WorkerQueueLength: 1000,
		CacheExpire:       500 * time.Millisecond,
		CacheClean:        1 * time.Second,
	}

	runner, err := NewRunner(logger, mr, conf)
	assert.NoError(t, err, "no error should be encountered during runner creation")

	payloads := []ocr2keepers.UpkeepPayload{
		{WorkID: "a"},
		{WorkID: "b"},
		{WorkID: "c"},
		{WorkID: "d"},
		{WorkID: "e"},
		{WorkID: "f"},
		{WorkID: "g"},
		{WorkID: "h"},
		{WorkID: "i"},
		{WorkID: "j"},
		{WorkID: "k"},
		{WorkID: "l"},
	}

	expected := make([]ocr2keepers.CheckResult, len(payloads))
	for i := range payloads {
		expected[i] = ocr2keepers.CheckResult{
			Payload: payloads[i],
		}
	}

	var wg sync.WaitGroup

	var tester func(w *sync.WaitGroup, m *mocks.MockRunnable, r *Runner, p []ocr2keepers.UpkeepPayload, e []ocr2keepers.CheckResult) = func(w *sync.WaitGroup, m *mocks.MockRunnable, r *Runner, p []ocr2keepers.UpkeepPayload, e []ocr2keepers.CheckResult) {
		m.On("CheckUpkeeps", append([]interface{}{mock.Anything}, toInterfaces(p...)...)...).Return(e, nil).Once().After(500 * time.Millisecond)

		// all batches should be collected into a single result set
		results, err := r.CheckUpkeeps(context.Background(), p...)

		assert.NoError(t, err, "no error should be encountered during upkeep checking")
		assert.Equal(t, e, results, "results should be returned without changes from the runnable")

		w.Done()
	}

	wg.Add(3)

	go tester(&wg, mr, runner, payloads[:5], expected[:5])
	go tester(&wg, mr, runner, payloads[5:10], expected[5:10])
	go tester(&wg, mr, runner, payloads[10:], expected[10:])

	wg.Wait()
}

func TestRunnerStartStop(t *testing.T) {
	logger := log.New(io.Discard, "", 0)
	mr := new(mocks.MockRunnable)

	conf := RunnerConfig{
		Workers:           2,
		WorkerQueueLength: 1000,
		CacheExpire:       500 * time.Millisecond,
		CacheClean:        1 * time.Second,
	}

	runner, err := NewRunner(logger, mr, conf)
	assert.NoError(t, err, "no error should be encountered during runner creation")

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		assert.NoError(t, runner.Start(context.Background()), "no error should be encountered during service start")
		wg.Done()
	}()

	// wait for the process to start
	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, runner.running.Load(), true, "process should be running")

	err = runner.Close()
	assert.NoError(t, err, "no error should be encountered during service stop")

	wg.Wait()

	assert.Equal(t, runner.running.Load(), false, "process should be running")
}

func TestRunnerErr(t *testing.T) {
	t.Run("Zero Length Payload No Error", func(t *testing.T) {
		logger := log.New(io.Discard, "", log.LstdFlags)
		mr := new(mocks.MockRunnable)

		conf := RunnerConfig{
			Workers:           2,
			WorkerQueueLength: 1000,
			CacheExpire:       500 * time.Millisecond,
			CacheClean:        1 * time.Second,
		}

		runner, err := NewRunner(logger, mr, conf)
		assert.NoError(t, err, "no error should be encountered during runner creation")

		payloads := []ocr2keepers.UpkeepPayload{}

		results, err := runner.CheckUpkeeps(context.Background(), payloads...)
		assert.NoError(t, err, "no error should be encountered during upkeep checking")
		assert.Len(t, results, 0, "result length should be zero without calling runnable")
	})

	t.Run("Multiple Runnable Errors Bubble Up", func(t *testing.T) {
		logger := log.New(io.Discard, "", log.LstdFlags)
		mr := new(mocks.MockRunnable)

		conf := RunnerConfig{
			Workers:           2,
			WorkerQueueLength: 1000,
			CacheExpire:       500 * time.Millisecond,
			CacheClean:        1 * time.Second,
		}

		runner, err := NewRunner(logger, mr, conf)
		assert.NoError(t, err, "no error should be encountered during runner creation")

		payloads := make([]ocr2keepers.UpkeepPayload, 20)
		for i := 0; i < 20; i++ {
			payloads[i] = ocr2keepers.UpkeepPayload{
				WorkID: fmt.Sprintf("id: %d", i),
			}
		}

		expected := make([]ocr2keepers.CheckResult, len(payloads))
		for i := range payloads {
			expected[i] = ocr2keepers.CheckResult{
				Payload: payloads[i],
			}
		}

		mVals := []interface{}{}
		for range payloads {
			mVals = append(mVals, mock.Anything)
		}

		mr.On("CheckUpkeeps", append([]interface{}{mock.Anything}, mVals...)...).Return(nil, fmt.Errorf("test error")).Times(2)

		results, err := runner.CheckUpkeeps(context.Background(), payloads...)

		assert.ErrorIs(t, err, ErrTooManyErrors, "runner should only return error when all runnable calls fail")
		assert.Len(t, results, 0, "result length should be zero")

		mr.AssertExpectations(t)
	})
}

func toInterfaces(payloads ...ocr2keepers.UpkeepPayload) []interface{} {
	asInter := []interface{}{}
	for i := range payloads {
		asInter = append(asInter, payloads[i])
	}
	return asInter
}
