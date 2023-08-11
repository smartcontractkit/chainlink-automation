package flows

import (
	"context"
	"io"
	"log"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg/v3/types"
)

// TODO placeholder
func TestConditionalTriggerFlows(t *testing.T) {
	conditional, err := ConditionalTriggerFlows(
		nil,
		nil,
		nil,
		&mockSubscriber{
			SubscribeFn: func() (int, chan ocr2keepers.BlockHistory, error) {
				return 0, nil, nil
			},
		},
		nil,
		nil,
		nil,
		&mockRunner{
			CheckUpkeepsFn: func(ctx context.Context, payload ...ocr2keepers.UpkeepPayload) ([]ocr2keepers.CheckResult, error) {
				return nil, nil
			},
		},
		nil,
		nil,
		nil,
		nil,
		log.New(io.Discard, "", 0),
	)
	assert.NotNil(t, conditional)
	assert.Nil(t, err)
}

// TODO placeholder
func TestLogTriggerFlows(t *testing.T) {
	logTrigger := LogTriggerFlows(
		nil,
		nil,
		nil,
		&mockRunner{
			CheckUpkeepsFn: func(ctx context.Context, payload ...ocr2keepers.UpkeepPayload) ([]ocr2keepers.CheckResult, error) {
				return nil, nil
			},
		},
		nil,
		nil,
		nil,
		time.Minute,
		time.Minute,
		nil,
		nil,
		nil,
		nil,
		log.New(io.Discard, "", 0),
	)
	assert.NotNil(t, logTrigger)
}

type mockRunner struct {
	CheckUpkeepsFn func(context.Context, ...ocr2keepers.UpkeepPayload) ([]ocr2keepers.CheckResult, error)
}

func (r *mockRunner) CheckUpkeeps(ctx context.Context, p ...ocr2keepers.UpkeepPayload) ([]ocr2keepers.CheckResult, error) {
	return r.CheckUpkeepsFn(ctx, p...)
}

type mockSubscriber struct {
	SubscribeFn   func() (int, chan ocr2keepers.BlockHistory, error)
	UnsubscribeFn func(int) error
}

func (r *mockSubscriber) Subscribe() (int, chan ocr2keepers.BlockHistory, error) {
	return r.SubscribeFn()
}
func (r *mockSubscriber) Unsubscribe(i int) error {
	return r.UnsubscribeFn(i)
}
