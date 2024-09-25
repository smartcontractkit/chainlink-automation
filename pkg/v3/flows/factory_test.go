package flows

import (
	"context"
	"io"
	"log"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	common "github.com/smartcontractkit/chainlink-common/pkg/types/automation"
)

func TestConditionalTriggerFlows(t *testing.T) {
	flows := NewConditionalTriggerFlows(
		nil,
		nil,
		nil,
		&mockSubscriber{
			SubscribeFn: func() (int, chan common.BlockHistory, error) {
				return 0, nil, nil
			},
		},
		nil,
		nil,
		nil,
		&mockRunner{
			CheckUpkeepsFn: func(ctx context.Context, payload ...common.UpkeepPayload) ([]common.CheckResult, error) {
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
	flows := NewLogTriggerFlows(
		nil,
		nil,
		nil,
		&mockRunner{
			CheckUpkeepsFn: func(ctx context.Context, payload ...common.UpkeepPayload) ([]common.CheckResult, error) {
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
	CheckUpkeepsFn func(context.Context, ...common.UpkeepPayload) ([]common.CheckResult, error)
}

func (r *mockRunner) CheckUpkeeps(ctx context.Context, p ...common.UpkeepPayload) ([]common.CheckResult, error) {
	return r.CheckUpkeepsFn(ctx, p...)
}

type mockSubscriber struct {
	SubscribeFn   func() (int, chan common.BlockHistory, error)
	UnsubscribeFn func(int) error
	StartFn       func(ctx context.Context) error
	CloseFn       func() error
}

func (r *mockSubscriber) Subscribe() (int, chan common.BlockHistory, error) {
	return r.SubscribeFn()
}
func (r *mockSubscriber) Unsubscribe(i int) error {
	return r.UnsubscribeFn(i)
}
func (r *mockSubscriber) Start(ctx context.Context) error {
	return r.StartFn(ctx)
}
func (r *mockSubscriber) Close() error {
	return r.CloseFn()
}
