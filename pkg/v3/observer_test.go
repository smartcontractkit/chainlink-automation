package ocr2keepers

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/tickers"
)

type mockTick struct {
	mock.Mock
}

func (m *mockTick) GetUpkeeps(ctx context.Context) ([]ocr2keepers.UpkeepPayload, error) {
	ret := m.Called(ctx)
	return ret.Get(0).([]ocr2keepers.UpkeepPayload), ret.Error(1)
}

type mockRunner struct {
	mock.Mock
}

func (m *mockRunner) CheckUpkeeps(ctx context.Context, payloads ...ocr2keepers.UpkeepPayload) ([]ocr2keepers.CheckResult, error) {
	var ret mock.Arguments
	if len(payloads) > 0 {
		ret = m.Called(ctx, payloads)
	} else {
		ret = m.Called(ctx)
	}

	return ret.Get(0).([]ocr2keepers.CheckResult), ret.Error(1)
}

type mockPreprocessor struct {
	mock.Mock
}

func (m *mockPreprocessor) PreProcess(ctx context.Context, payloads []ocr2keepers.UpkeepPayload) ([]ocr2keepers.UpkeepPayload, error) {
	ret := m.Called(ctx, payloads)
	return ret.Get(0).([]ocr2keepers.UpkeepPayload), ret.Error(1)
}

type mockPostprocessor struct {
	mock.Mock
}

func (m *mockPostprocessor) PostProcess(ctx context.Context, results []ocr2keepers.CheckResult) error {
	ret := m.Called(ctx, results)
	return ret.Error(0)
}

func TestNewObserver(t *testing.T) {
	type args struct {
		preprocessors []PreProcessor
		postprocessor PostProcessor
		runner        Runner
	}
	tests := []struct {
		name string
		args args
		want Observer
	}{
		{
			name: "should return an Observer",
			args: args{
				preprocessors: []PreProcessor{new(mockPreprocessor)},
				postprocessor: new(mockPostprocessor),
				runner:        new(mockRunner),
			},
			want: Observer{
				Preprocessors: []PreProcessor{new(mockPreprocessor)},
				Postprocessor: new(mockPostprocessor),
				Runner:        new(mockRunner),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, *NewObserver(tt.args.preprocessors, tt.args.postprocessor, tt.args.runner), "NewObserver(%v, %v, %v)", tt.args.preprocessors, tt.args.postprocessor, tt.args.runner)
		})
	}
}

func TestObserve_Process(t *testing.T) {
	type fields struct {
		Preprocessors []PreProcessor
		Postprocessor PostProcessor
		Runner        Runner
	}
	type args struct {
		ctx  context.Context
		tick tickers.Tick
	}
	type expectations struct {
		tickReturn         []ocr2keepers.UpkeepPayload
		tickErr            error
		runnerReturn       []ocr2keepers.CheckResult
		runnerErr          error
		preprocessorReturn []ocr2keepers.UpkeepPayload
		preprocessorErr    error
		postprocessorErr   error
	}
	expectedPayload := []ocr2keepers.UpkeepPayload{}
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
				Preprocessors: []PreProcessor{new(mockPreprocessor)},
				Postprocessor: new(mockPostprocessor),
				Runner:        new(mockRunner),
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
				Preprocessors: []PreProcessor{new(mockPreprocessor)},
				Postprocessor: new(mockPostprocessor),
				Runner:        new(mockRunner),
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
				Preprocessors: []PreProcessor{new(mockPreprocessor)},
				Postprocessor: new(mockPostprocessor),
				Runner:        new(mockRunner),
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
				Preprocessors: []PreProcessor{new(mockPreprocessor)},
				Postprocessor: new(mockPostprocessor),
				Runner:        new(mockRunner),
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
				Preprocessors: []PreProcessor{new(mockPreprocessor)},
				Postprocessor: new(mockPostprocessor),
				Runner:        new(mockRunner),
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
			o := &Observer{
				Preprocessors: tt.fields.Preprocessors,
				Postprocessor: tt.fields.Postprocessor,
				Runner:        tt.fields.Runner,
			}
			// Mock calls
			tt.args.tick.(*mockTick).On("GetUpkeeps", tt.args.ctx).Return(expectedPayload, tt.expectations.tickErr)
			for i := range tt.fields.Preprocessors {
				tt.fields.Preprocessors[i].(*mockPreprocessor).On("PreProcess", tt.args.ctx, expectedPayload).Return(expectedPayload, tt.expectations.preprocessorErr)
			}

			vals := []interface{}{}
			vals = append(vals, tt.args.ctx)
			for i := range expectedPayload {
				vals = append(vals, expectedPayload[i])
			}

			tt.fields.Runner.(*mockRunner).On("CheckUpkeeps", vals...).Return(expectedCheckResults, tt.expectations.runnerErr)
			tt.fields.Postprocessor.(*mockPostprocessor).On("PostProcess", tt.args.ctx, expectedCheckResults).Return(tt.expectations.postprocessorErr)

			tt.wantErr(t, o.Process(tt.args.ctx, tt.args.tick), fmt.Sprintf("Process(%v, %v)", tt.args.ctx, tt.args.tick))
		})
	}
}
