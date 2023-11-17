package ocr2keepers

import (
	"context"
	"fmt"
	"io"
	"log"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/smartcontractkit/chainlink-automation/pkg/v3/tickers"
	ocr2keepers "github.com/smartcontractkit/chainlink-automation/pkg/v3/types"
)

type mockTick struct {
	mock.Mock
}

func (m *mockTick) Value(ctx context.Context) ([]int, error) {
	ret := m.Called(ctx)
	return ret.Get(0).([]int), ret.Error(1)
}

type mockProcessFunc struct {
	mock.Mock
}

func (m *mockProcessFunc) Process(ctx context.Context, values ...int) ([]ocr2keepers.CheckResult, error) {
	var ret mock.Arguments
	if len(values) > 0 {
		ret = m.Called(ctx, values)
	} else {
		ret = m.Called(ctx)
	}

	return ret.Get(0).([]ocr2keepers.CheckResult), ret.Error(1)
}

type mockPreprocessor struct {
	mock.Mock
}

func (m *mockPreprocessor) PreProcess(ctx context.Context, values []int) ([]int, error) {
	ret := m.Called(ctx, values)
	return ret.Get(0).([]int), ret.Error(1)
}

type mockPostprocessor struct {
	mock.Mock
}

func (m *mockPostprocessor) PostProcess(ctx context.Context, results []ocr2keepers.CheckResult, payloads []int) error {
	ret := m.Called(ctx, results)
	return ret.Error(0)
}

func TestNewGenericObserver(t *testing.T) {
	t.Skip()

	type args struct {
		preprocessors []PreProcessor[int]
		postprocessor PostProcessor[int]
		runner        func(context.Context, ...int) ([]ocr2keepers.CheckResult, error)
		limit         time.Duration
		logger        *log.Logger
	}

	tests := []struct {
		name string
		args args
		want Observer[int]
	}{
		{
			name: "should return an Observer",
			args: args{
				preprocessors: []PreProcessor[int]{new(mockPreprocessor)},
				postprocessor: new(mockPostprocessor),
				runner:        new(mockProcessFunc).Process,
				limit:         50 * time.Millisecond,
				logger:        log.New(io.Discard, "", 0),
			},
			want: Observer[int]{
				Preprocessors:    []PreProcessor[int]{new(mockPreprocessor)},
				Postprocessor:    new(mockPostprocessor),
				processFunc:      new(mockProcessFunc).Process,
				processTimeLimit: 50 * time.Millisecond,
				lggr:             log.New(io.Discard, "", 0),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			want := Observer[int]{
				Preprocessors:    tt.args.preprocessors,
				Postprocessor:    tt.args.postprocessor,
				processFunc:      tt.args.runner,
				processTimeLimit: tt.args.limit,
			}

			assert.Equalf(
				t,
				want,
				*NewGenericObserver(tt.args.preprocessors, tt.args.postprocessor, tt.args.runner, 50*time.Millisecond, tt.args.logger),
				"NewObserver(%v, %v, %v)",
				tt.args.preprocessors,
				tt.args.postprocessor,
				tt.args.runner,
			)
		})
	}
}

func TestObserve_Process(t *testing.T) {
	type fields struct {
		Preprocessors []PreProcessor[int]
		Postprocessor PostProcessor[int]
		Processor     *mockProcessFunc
	}

	type args struct {
		ctx  context.Context
		tick tickers.Tick[[]int]
	}

	type expectations struct {
		tickReturn         []int
		tickErr            error
		runnerReturn       []ocr2keepers.CheckResult
		runnerErr          error
		preprocessorReturn []int
		preprocessorErr    error
		postprocessorErr   error
	}

	expectedPayload := []int{}
	expectedCheckResults := []ocr2keepers.CheckResult{}
	tests := []struct {
		name         string
		fields       fields
		args         args
		expectations expectations
		wantErr      assert.ErrorAssertionFunc
	}{
		{
			name: "should return an error if tick.GetUpkeeps returns an error",
			fields: fields{
				Preprocessors: []PreProcessor[int]{new(mockPreprocessor)},
				Postprocessor: new(mockPostprocessor),
				Processor:     new(mockProcessFunc),
			},
			args: args{
				ctx:  context.Background(),
				tick: new(mockTick),
			},
			expectations: expectations{
				tickReturn:         expectedPayload,
				tickErr:            fmt.Errorf("tick.GetUpkeeps error"),
				runnerReturn:       expectedCheckResults,
				runnerErr:          nil,
				preprocessorReturn: expectedPayload,
				preprocessorErr:    nil,
				postprocessorErr:   nil,
			},
			wantErr: assert.Error,
		},
		{
			name: "should return an error if preprocessor.PreProcess returns an error",
			fields: fields{
				Preprocessors: []PreProcessor[int]{new(mockPreprocessor)},
				Postprocessor: new(mockPostprocessor),
				Processor:     new(mockProcessFunc),
			},
			args: args{
				ctx:  context.Background(),
				tick: new(mockTick),
			},
			expectations: expectations{
				tickReturn:         expectedPayload,
				tickErr:            nil,
				runnerReturn:       expectedCheckResults,
				runnerErr:          nil,
				preprocessorReturn: expectedPayload,
				preprocessorErr:    fmt.Errorf("preprocessor.PreProcess error"),
				postprocessorErr:   nil,
			},
			wantErr: assert.Error,
		},
		{
			name: "should return an error if runner.CheckUpkeeps returns an error",
			fields: fields{
				Preprocessors: []PreProcessor[int]{new(mockPreprocessor)},
				Postprocessor: new(mockPostprocessor),
				Processor:     new(mockProcessFunc),
			},
			args: args{
				ctx:  context.Background(),
				tick: new(mockTick),
			},
			expectations: expectations{
				tickReturn:         expectedPayload,
				tickErr:            nil,
				runnerReturn:       expectedCheckResults,
				runnerErr:          fmt.Errorf("runner.CheckUpkeeps error"),
				preprocessorReturn: expectedPayload,
				preprocessorErr:    nil,
				postprocessorErr:   nil,
			},
			wantErr: assert.Error,
		},
		{
			name: "should return an error if postprocessor.PostProcess returns an error",
			fields: fields{
				Preprocessors: []PreProcessor[int]{new(mockPreprocessor)},
				Postprocessor: new(mockPostprocessor),
				Processor:     new(mockProcessFunc),
			},
			args: args{
				ctx:  context.Background(),
				tick: new(mockTick),
			},
			expectations: expectations{
				tickReturn:         expectedPayload,
				tickErr:            nil,
				runnerReturn:       expectedCheckResults,
				runnerErr:          nil,
				preprocessorReturn: expectedPayload,
				preprocessorErr:    nil,
				postprocessorErr:   fmt.Errorf("postprocessor.PostProcess error"),
			},
			wantErr: assert.Error,
		},
		{
			name: "should return nil if all steps succeed",
			fields: fields{
				Preprocessors: []PreProcessor[int]{new(mockPreprocessor)},
				Postprocessor: new(mockPostprocessor),
				Processor:     new(mockProcessFunc),
			},
			args: args{
				ctx:  context.Background(),
				tick: new(mockTick),
			},
			expectations: expectations{
				tickReturn:         expectedPayload,
				tickErr:            nil,
				runnerReturn:       expectedCheckResults,
				runnerErr:          nil,
				preprocessorReturn: expectedPayload,
				preprocessorErr:    nil,
				postprocessorErr:   nil,
			},
			wantErr: assert.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &Observer[int]{
				lggr:             log.New(io.Discard, "", 0),
				Preprocessors:    tt.fields.Preprocessors,
				Postprocessor:    tt.fields.Postprocessor,
				processFunc:      tt.fields.Processor.Process,
				processTimeLimit: 50 * time.Millisecond,
			}

			// Mock calls
			tt.args.tick.(*mockTick).On("Value", mock.AnythingOfType("*context.timerCtx")).Return(expectedPayload, tt.expectations.tickErr)
			for i := range tt.fields.Preprocessors {
				tt.fields.Preprocessors[i].(*mockPreprocessor).On("PreProcess", mock.AnythingOfType("*context.timerCtx"), expectedPayload).Return(expectedPayload, tt.expectations.preprocessorErr)
			}

			vals := []interface{}{}
			vals = append(vals, mock.AnythingOfType("*context.timerCtx"))
			for i := range expectedPayload {
				vals = append(vals, expectedPayload[i])
			}

			tt.fields.Processor.On("Process", vals...).Return(expectedCheckResults, tt.expectations.runnerErr)
			tt.fields.Postprocessor.(*mockPostprocessor).On("PostProcess", mock.AnythingOfType("*context.timerCtx"), expectedCheckResults).Return(tt.expectations.postprocessorErr)

			tt.wantErr(t, o.Process(tt.args.ctx, tt.args.tick), fmt.Sprintf("Process(%v, %v)", tt.args.ctx, tt.args.tick))
		})
	}
}

type mockSlowProcessFunc struct {
}

func (m *mockSlowProcessFunc) Process(ctx context.Context, values ...int) ([]ocr2keepers.CheckResult, error) {
	// Wait a bit to simulate a slow process
	<-time.After(500 * time.Millisecond)
	return []ocr2keepers.CheckResult{}, nil
}

func TestObserve_ConcurrentProcess(t *testing.T) {
	// test that multiple calls to the observer.Process can be run concurrently with --race flag on
	// Assumes that preProcessor, PostProcessor and ProcessFunc are all thread safe
	ctx := context.Background()
	expectedPayload := []int{1, 2, 3}
	expectedCheckResults := []ocr2keepers.CheckResult{}
	pre := new(mockPreprocessor)
	pre.On("PreProcess", mock.Anything, expectedPayload).Return(expectedPayload, nil).Times(3)
	post := new(mockPostprocessor)
	post.On("PostProcess", mock.Anything, expectedCheckResults).Return(nil).Times(3)

	o := &Observer[int]{
		lggr:             log.New(io.Discard, "", 0),
		Preprocessors:    []PreProcessor[int]{pre},
		Postprocessor:    post,
		processFunc:      new(mockSlowProcessFunc).Process,
		processTimeLimit: 2 * time.Second,
	}

	var wg sync.WaitGroup

	var tester func(w *sync.WaitGroup) = func(w *sync.WaitGroup) {
		tick := new(mockTick)
		tick.On("Value", mock.Anything).Return(expectedPayload, nil).Times(3)

		err := o.Process(ctx, tick)
		assert.NoError(t, err, "no error should be encountered during upkeep checking")

		w.Done()
	}

	wg.Add(3)

	go tester(&wg)
	go tester(&wg)
	go tester(&wg)

	wg.Wait()
}
