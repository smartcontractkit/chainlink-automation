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
	flows := ConditionalTriggerFlows(
		nil,
		nil,
		nil,
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
