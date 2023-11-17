package flows

import (
	"context"
	"io"
	"log"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	ocr2keepers "github.com/smartcontractkit/chainlink-automation/pkg/v3/types"
)

func TestConditionalTriggerFlows(t *testing.T) {
	flows := ConditionalTriggerFlows(
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
		log.New(io.Discard, "", 0),
	)
	assert.Equal(t, 2, len(flows))
}

func TestLogTriggerFlows(t *testing.T) {
	flows := LogTriggerFlows(
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
		time.Minute,
		nil,
		nil,
		nil,
		log.New(io.Discard, "", 0),
	)
	assert.Equal(t, 3, len(flows))
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
