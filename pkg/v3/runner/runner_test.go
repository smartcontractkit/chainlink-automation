package runner

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	ocr2keepers "github.com/smartcontractkit/chainlink-common/pkg/types/automation"

	"github.com/smartcontractkit/chainlink-automation/pkg/v3/types/mocks"
)

var (
	result1 = ocr2keepers.CheckResult{
		Retryable: false,
		UpkeepID:  ocr2keepers.UpkeepIdentifier([32]byte{1}),
		Trigger: ocr2keepers.Trigger{
			BlockNumber: 1,
			BlockHash:   [32]byte{1},
		},
		WorkID: "workID1",
	}
	result2 = ocr2keepers.CheckResult{
		Retryable: false,
		UpkeepID:  ocr2keepers.UpkeepIdentifier([32]byte{2}),
		Trigger: ocr2keepers.Trigger{
			BlockNumber: 2,
			BlockHash:   [32]byte{2},
		},
		WorkID: "workID2",
	}
	result3 = ocr2keepers.CheckResult{
		Retryable: false,
		UpkeepID:  ocr2keepers.UpkeepIdentifier([32]byte{3}),
		Trigger: ocr2keepers.Trigger{
			BlockNumber: 3,
			BlockHash:   [32]byte{3},
		},
		WorkID: "workID3",
	}
	result4 = ocr2keepers.CheckResult{
		Retryable: false,
		UpkeepID:  ocr2keepers.UpkeepIdentifier([32]byte{4}),
		Trigger: ocr2keepers.Trigger{
			BlockNumber: 4,
			BlockHash:   [32]byte{4},
		},
		WorkID: "workID4",
	}
	result5 = ocr2keepers.CheckResult{
		Retryable: false,
		UpkeepID:  ocr2keepers.UpkeepIdentifier([32]byte{5}),
		Trigger: ocr2keepers.Trigger{
			BlockNumber: 5,
			BlockHash:   [32]byte{5},
		},
		WorkID: "workID5",
	}
)

func TestRunnerCache(t *testing.T) {
	logger := log.New(io.Discard, "", 0)

	conf := RunnerConfig{
		Workers:           2,
		WorkerQueueLength: 1000,
		CacheExpire:       500 * time.Millisecond,
		CacheClean:        1 * time.Second,
	}

	payloads := []ocr2keepers.UpkeepPayload{
		{
			UpkeepID: result1.UpkeepID,
			Trigger:  result1.Trigger,
			WorkID:   "workID1",
		},
		{
			UpkeepID: result2.UpkeepID,
			Trigger:  result2.Trigger,
			WorkID:   "workID2",
		},
		{
			UpkeepID: result3.UpkeepID,
			Trigger:  result3.Trigger,
			WorkID:   "workID3",
		},
		{
			UpkeepID: result4.UpkeepID,
			Trigger:  result4.Trigger,
			WorkID:   "workID4",
		},
		{
			UpkeepID: result5.UpkeepID,
			Trigger:  result5.Trigger,
			WorkID:   "workID5",
		},
	}

	expected := make([]ocr2keepers.CheckResult, len(payloads))
	for i := range payloads {
		expected[i] = ocr2keepers.CheckResult{
			UpkeepID: payloads[i].UpkeepID,
			Trigger:  payloads[i].Trigger,
			WorkID:   payloads[i].WorkID,
		}
	}

	count := atomic.Int32{}
	mr := &mockRunnable{
		CheckUpkeepsFn: func(ctx context.Context, payload ...ocr2keepers.UpkeepPayload) ([]ocr2keepers.CheckResult, error) {
			count.Add(1)
			time.Sleep(500 * time.Millisecond)
			return expected, nil
		},
	}

	runner, err := NewRunner(logger, mr, conf)
	assert.NoError(t, err, "no error should be encountered during runner creation")

	results, err := runner.CheckUpkeeps(context.Background(), payloads...)
	assert.NoError(t, err, "no error should be encountered during upkeep checking")
	assert.Equal(t, expected, results, "results should be returned without changes from the runnable")

	// ensure that a call with the same payloads uses cache instead of calling runnable
	results, err = runner.CheckUpkeeps(context.Background(), payloads...)
	assert.NoError(t, err, "no error should be encountered during upkeep checking")
	assert.Equal(t, expected, results, "results should be returned without changes from the runnable")
	assert.Equal(t, int32(1), count.Load())
}

func TestRunnerCacheDifferentTriggerBlock(t *testing.T) {
	logger := log.New(io.Discard, "", 0)

	conf := RunnerConfig{
		Workers:           2,
		WorkerQueueLength: 1000,
		CacheExpire:       5 * time.Second,
		CacheClean:        5 * time.Second,
	}

	payloads := []ocr2keepers.UpkeepPayload{
		{
			UpkeepID: ocr2keepers.UpkeepIdentifier([32]byte{1}),
			Trigger: ocr2keepers.Trigger{
				BlockNumber: 1,
				BlockHash:   [32]byte{1},
			},
			WorkID: "workID1",
		},
	}

	newerBlockPayloads := []ocr2keepers.UpkeepPayload{
		{
			UpkeepID: ocr2keepers.UpkeepIdentifier([32]byte{1}),
			Trigger: ocr2keepers.Trigger{
				BlockNumber: 2,
				BlockHash:   [32]byte{1},
			},
			WorkID: "workID1",
		},
	}

	expected := make([]ocr2keepers.CheckResult, len(payloads))
	for i := range payloads {
		expected[i] = ocr2keepers.CheckResult{
			UpkeepID: payloads[i].UpkeepID,
			Trigger:  payloads[i].Trigger,
			WorkID:   payloads[i].WorkID,
		}
	}
	newerExpected := make([]ocr2keepers.CheckResult, len(newerBlockPayloads))
	for i := range newerBlockPayloads {
		newerExpected[i] = ocr2keepers.CheckResult{
			UpkeepID: newerBlockPayloads[i].UpkeepID,
			Trigger:  newerBlockPayloads[i].Trigger,
			WorkID:   newerBlockPayloads[i].WorkID,
		}
	}

	count := atomic.Int32{}
	mr := &mockRunnable{
		CheckUpkeepsFn: func(ctx context.Context, payload ...ocr2keepers.UpkeepPayload) ([]ocr2keepers.CheckResult, error) {
			time.Sleep(500 * time.Millisecond)
			count.Add(1)
			if count.Load() == int32(1) {
				return expected, nil
			} else if count.Load() == int32(2) {
				return newerExpected, nil
			}
			return nil, errors.New("unexpected call")
		},
	}
	runner, err := NewRunner(logger, mr, conf)
	assert.NoError(t, err, "no error should be encountered during runner creation")

	// ensure that context and payloads are passed through to the runnable
	// return results that should be cached
	results, err := runner.CheckUpkeeps(context.Background(), payloads...)
	assert.NoError(t, err, "no error should be encountered during upkeep checking")
	assert.Equal(t, expected, results, "results should be returned without changes from the runnable")

	// ensure that a call with the same workID but different trigger block causes a new call to CheckUpkeeps
	results, err = runner.CheckUpkeeps(context.Background(), newerBlockPayloads...)
	assert.NoError(t, err, "no error should be encountered during upkeep checking")
	assert.Equal(t, newerExpected, results, "results should be returned without changes from the runnable")

	// ensure that the higher checkBlock overwrote the cache, so another call with same payload does not call CheckUpkeeps
	results, err = runner.CheckUpkeeps(context.Background(), newerBlockPayloads...)
	assert.NoError(t, err, "no error should be encountered during upkeep checking")
	assert.Equal(t, newerExpected, results, "results should be returned without changes from the runnable")

	assert.Equal(t, int32(2), count.Load())
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
			UpkeepID: payloads[i].UpkeepID,
			Trigger:  payloads[i].Trigger,
		}
	}

	// ensure that context and payloads are passed through to the runnable
	// payloads and results should be split by batches
	mr.On("CheckUpkeeps", append([]interface{}{mock.Anything}, toInterfaces(payloads[:10]...)...)...).Return(expected[:10], nil).Once().After(500 * time.Millisecond)
	mr.On("CheckUpkeeps", append([]interface{}{mock.Anything}, toInterfaces(payloads[10:]...)...)...).Return(expected[10:], nil).Once().After(500 * time.Millisecond)

	// all batches should be collected into a single result set
	results, err := runner.CheckUpkeeps(context.Background(), payloads...)

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
			UpkeepID: payloads[i].UpkeepID,
			Trigger:  payloads[i].Trigger,
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

		count := atomic.Int32{}
		mr := &mockRunnable{
			CheckUpkeepsFn: func(ctx context.Context, payload ...ocr2keepers.UpkeepPayload) ([]ocr2keepers.CheckResult, error) {
				count.Add(1)
				return nil, fmt.Errorf("test error")
			},
		}

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
				UpkeepID: payloads[i].UpkeepID,
				Trigger:  payloads[i].Trigger,
			}
		}

		results, err := runner.CheckUpkeeps(context.Background(), payloads...)

		assert.ErrorIs(t, err, ErrTooManyErrors, "runner should only return error when all runnable calls fail")
		assert.Len(t, results, 0, "result length should be zero")
		assert.Equal(t, int32(2), count.Load())
	})
}

func toInterfaces(payloads ...ocr2keepers.UpkeepPayload) []interface{} {
	asInter := []interface{}{}
	for i := range payloads {
		asInter = append(asInter, payloads[i])
	}
	return asInter
}

type mockRunnable struct {
	CheckUpkeepsFn func(context.Context, ...ocr2keepers.UpkeepPayload) ([]ocr2keepers.CheckResult, error)
}

func (r *mockRunnable) CheckUpkeeps(ctx context.Context, payloads ...ocr2keepers.UpkeepPayload) ([]ocr2keepers.CheckResult, error) {
	return r.CheckUpkeepsFn(ctx, payloads...)
}
