package observer

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/smartcontractkit/ocr2keepers/pkg/types"
	"github.com/stretchr/testify/require"
)

func TestRegisterObservers(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	tick := time.Millisecond * 2
	o1, o2 := new(mockObserver), new(mockObserver)
	tk := time.NewTicker(tick)

	time.AfterFunc(time.Duration(tick*2), func() {
		cancel()
		require.Greater(t, o1.getCount(), int32(0))
		require.Greater(t, o2.getCount(), int32(0))
	})

	RegisterObservers[time.Time](ctx, tk.C, o1, o2)
}

type mockObserver struct {
	processCount int32
}

func (o *mockObserver) Process(ctx context.Context, t time.Time) {
	atomic.AddInt32(&o.processCount, 1)
}

func (o *mockObserver) Propose(ctx context.Context) ([]types.UpkeepResult, error) {
	return nil, nil
}

func (o *mockObserver) getCount() int32 {
	return atomic.LoadInt32(&o.processCount)
}
