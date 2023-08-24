package postprocessors

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/smartcontractkit/ocr2keepers/pkg/v3/types"
)

func TestNewIneligiblePostProcessor(t *testing.T) {
	for _, tc := range []struct {
		name         string
		stateUpdater types.UpkeepStateUpdater
		results      []types.CheckResult
		wantLog      string
		expectsErr   bool
		wantErr      error
	}{
		{
			name: "upkeep state is updated as per one check result",
			stateUpdater: &mockStateUpdater{
				SetUpkeepStateFn: func(ctx context.Context, result types.CheckResult, state types.UpkeepState) error {
					return nil
				},
			},
			results: []types.CheckResult{
				{
					PipelineExecutionState: 0,
					Eligible:               false,
				},
			},
			wantLog: "post-processing 1 results, 1 ineligible",
		},
		{
			name: "upkeep state is updated as per multiple check results",
			stateUpdater: &mockStateUpdater{
				SetUpkeepStateFn: func(ctx context.Context, result types.CheckResult, state types.UpkeepState) error {
					return nil
				},
			},
			results: []types.CheckResult{
				{
					PipelineExecutionState: 0,
					Eligible:               false,
				},
				{
					PipelineExecutionState: 0,
					Eligible:               false,
				},
				{
					PipelineExecutionState: 0,
					Eligible:               false,
				},
			},
			wantLog: "post-processing 3 results, 3 ineligible",
		},
		{
			name: "upkeep state errors",
			stateUpdater: &mockStateUpdater{
				SetUpkeepStateFn: func(ctx context.Context, result types.CheckResult, state types.UpkeepState) error {
					return errors.New("upkeep state boom")
				},
			},
			results: []types.CheckResult{
				{
					PipelineExecutionState: 0,
					Eligible:               false,
				},
				{
					PipelineExecutionState: 0,
					Eligible:               true,
				},
				{
					PipelineExecutionState: 0,
					Eligible:               false,
				},
			},
			expectsErr: true,
			wantErr: errors.New(`upkeep state boom
upkeep state boom`),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			l := log.Default()
			l.SetOutput(&buf)
			processor := NewIneligiblePostProcessor(tc.stateUpdater, l)

			err := processor.PostProcess(context.Background(), tc.results, nil)
			if tc.expectsErr {
				assert.Error(t, err)
				assert.Equal(t, tc.wantErr.Error(), err.Error())
			} else {
				assert.NoError(t, err)
				fmt.Println(buf.String())
				assert.True(t, strings.Contains(buf.String(), tc.wantLog))
			}
		})
	}
}

type mockStateUpdater struct {
	types.UpkeepStateUpdater
	SetUpkeepStateFn func(context.Context, types.CheckResult, types.UpkeepState) error
}

func (u *mockStateUpdater) SetUpkeepState(ctx context.Context, res types.CheckResult, state types.UpkeepState) error {
	return u.SetUpkeepStateFn(ctx, res, state)
}
