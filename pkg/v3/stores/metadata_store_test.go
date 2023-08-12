package stores

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/smartcontractkit/ocr2keepers/pkg/v3/tickers"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/types"
)

func TestNewMetadataStore(t *testing.T) {
	t.Run("the metadata store starts, reads from the block ticker, and stops without erroring", func(t *testing.T) {
		canClose := make(chan struct{}, 1)
		finished := make(chan struct{}, 1)
		blockSubscriber := &mockBlockSubscriber{
			ch: make(chan types.BlockHistory),
		}

		go func() {
			blockSubscriber.ch <- types.BlockHistory{
				types.BlockKey{
					Number: 4,
				},
				types.BlockKey{
					Number: 3,
				},
			}
		}()

		blockTicker, err := tickers.NewBlockTicker(blockSubscriber)
		assert.NoError(t, err)
		go func() {
			err2 := blockTicker.Start(context.Background())
			assert.NoError(t, err2)
		}()

		store := NewMetadataStore(blockTicker, nil)

		go func() {
			err = store.Start(context.Background())
			assert.NoError(t, err)
			finished <- struct{}{}
		}()

		go func() {
			for {
				if len(store.GetBlockHistory()) == 2 {
					canClose <- struct{}{}
					return
				}
				time.Sleep(time.Second)
			}
		}()

		<-canClose

		closeErr := store.Close()
		assert.NoError(t, closeErr)

		<-finished
	})

	t.Run("the metadata store starts, reads from the block ticker, and stops via a cancelled context without erroring", func(t *testing.T) {
		canClose := make(chan struct{}, 1)
		finished := make(chan struct{}, 1)
		blockSubscriber := &mockBlockSubscriber{
			ch: make(chan types.BlockHistory),
		}

		go func() {
			blockSubscriber.ch <- types.BlockHistory{
				types.BlockKey{
					Number: 4,
				},
				types.BlockKey{
					Number: 3,
				},
			}
		}()

		blockTicker, err := tickers.NewBlockTicker(blockSubscriber)
		assert.NoError(t, err)

		go func() {
			err2 := blockTicker.Start(context.Background())
			assert.NoError(t, err2)
		}()

		store := NewMetadataStore(blockTicker, nil)

		ctx, cancelFn := context.WithCancel(context.Background())
		go func() {
			err = store.Start(ctx)
			assert.NoError(t, err)
			finished <- struct{}{}
		}()

		go func() {
			for {
				if len(store.GetBlockHistory()) == 2 {
					canClose <- struct{}{}
					return
				}
				time.Sleep(time.Second)
			}
		}()

		<-canClose

		cancelFn()

		<-finished
	})

	t.Run("starting an already started metadata store returns an error", func(t *testing.T) {
		store := NewMetadataStore(nil, nil)
		store.running.Store(true)
		err := store.Start(context.Background())
		assert.Error(t, err)
	})

	t.Run("closing an already closed metadata store returns an error", func(t *testing.T) {
		store := NewMetadataStore(nil, nil)
		store.running.Store(false)
		err := store.Close()
		assert.Error(t, err)
	})
}

func TestMetadataStore_AddConditionalProposal(t *testing.T) {
	for _, tc := range []struct {
		name            string
		addProposals    [][]types.CoordinatedBlockProposal
		afterAdd        []types.CoordinatedBlockProposal
		deleteProposals []types.CoordinatedBlockProposal
		afterDelete     []types.CoordinatedBlockProposal
		timeFn          func() time.Time
	}{
		{
			name: "all unique proposals are added and retrieved, existent keys are successfully deleted",
			addProposals: [][]types.CoordinatedBlockProposal{
				{
					{
						WorkID: "workID1",
					},
					{
						WorkID: "workID2",
					},
				},
				{
					{
						WorkID: "workID3",
					},
					{
						WorkID: "workID4",
					},
				},
			},
			afterAdd: []types.CoordinatedBlockProposal{
				{
					WorkID: "workID1",
				},
				{
					WorkID: "workID2",
				},
				{
					WorkID: "workID3",
				},
				{
					WorkID: "workID4",
				},
			},
			deleteProposals: []types.CoordinatedBlockProposal{
				{
					WorkID: "workID1",
				},
				{
					WorkID: "workID3",
				},
				{
					WorkID: "workID5",
				},
			},
			afterDelete: []types.CoordinatedBlockProposal{
				{
					WorkID: "workID2",
				},
				{
					WorkID: "workID4",
				},
			},
			timeFn: time.Now,
		},
		{
			name: "duplicate proposals aren't returned, existent keys are successfully deleted",
			addProposals: [][]types.CoordinatedBlockProposal{
				{
					{
						WorkID: "workID1",
					},
					{
						WorkID: "workID1",
					},
				},
				{
					{
						WorkID: "workID1",
					},
					{
						WorkID: "workID1",
					},
				},
			},
			afterAdd: []types.CoordinatedBlockProposal{
				{
					WorkID: "workID1",
				},
			},
			deleteProposals: []types.CoordinatedBlockProposal{
				{
					WorkID: "workID1",
				},
			},
			afterDelete: []types.CoordinatedBlockProposal{},
			timeFn:      time.Now,
		},
		{
			name: "proposals added three days ago aren't returned, non existent keys result in a no op delete",
			addProposals: [][]types.CoordinatedBlockProposal{
				{
					{
						WorkID: "workID1",
					},
					{
						WorkID: "workID1",
					},
				},
				{
					{
						WorkID: "workID1",
					},
					{
						WorkID: "workID1",
					},
				},
			},
			afterAdd: []types.CoordinatedBlockProposal{},
			deleteProposals: []types.CoordinatedBlockProposal{
				{
					WorkID: "workID1",
				},
			},
			afterDelete: []types.CoordinatedBlockProposal{},
			timeFn: func() time.Time {
				return time.Now().Add(-72 * time.Hour)
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			oldTimeFn := timeFn
			timeFn = tc.timeFn
			defer func() {
				timeFn = oldTimeFn
			}()

			store := NewMetadataStore(nil, nil)
			for _, proposal := range tc.addProposals {
				store.AddConditionalProposal(proposal...)
			}
			proposals := store.ViewConditionalProposal()
			assert.Equal(t, tc.afterAdd, proposals)
			store.RemoveConditionalProposal(tc.deleteProposals...)
			proposals = store.ViewConditionalProposal()
			assert.Equal(t, tc.afterDelete, proposals)
		})

	}
}

func TestMetadataStore_AddLogRecoveryProposal(t *testing.T) {
	for _, tc := range []struct {
		name            string
		addProposals    [][]types.CoordinatedBlockProposal
		afterAdd        []types.CoordinatedBlockProposal
		deleteProposals []types.CoordinatedBlockProposal
		afterDelete     []types.CoordinatedBlockProposal
		timeFn          func() time.Time
	}{
		{
			name: "all unique proposals are added and retrieved, existent keys are successfully deleted",
			addProposals: [][]types.CoordinatedBlockProposal{
				{
					{
						WorkID: "workID1",
					},
					{
						WorkID: "workID2",
					},
				},
				{
					{
						WorkID: "workID3",
					},
					{
						WorkID: "workID4",
					},
				},
			},
			afterAdd: []types.CoordinatedBlockProposal{
				{
					WorkID: "workID1",
				},
				{
					WorkID: "workID2",
				},
				{
					WorkID: "workID3",
				},
				{
					WorkID: "workID4",
				},
			},
			deleteProposals: []types.CoordinatedBlockProposal{
				{
					WorkID: "workID1",
				},
				{
					WorkID: "workID3",
				},
				{
					WorkID: "workID5",
				},
			},
			afterDelete: []types.CoordinatedBlockProposal{
				{
					WorkID: "workID2",
				},
				{
					WorkID: "workID4",
				},
			},
			timeFn: time.Now,
		},
		{
			name: "duplicate proposals aren't returned, existent keys are successfully deleted",
			addProposals: [][]types.CoordinatedBlockProposal{
				{
					{
						WorkID: "workID1",
					},
					{
						WorkID: "workID1",
					},
				},
				{
					{
						WorkID: "workID1",
					},
					{
						WorkID: "workID1",
					},
				},
			},
			afterAdd: []types.CoordinatedBlockProposal{
				{
					WorkID: "workID1",
				},
			},
			deleteProposals: []types.CoordinatedBlockProposal{
				{
					WorkID: "workID1",
				},
			},
			afterDelete: []types.CoordinatedBlockProposal{},
			timeFn:      time.Now,
		},
		{
			name: "proposals added three days ago aren't returned, non existent keys result in a no op delete",
			addProposals: [][]types.CoordinatedBlockProposal{
				{
					{
						WorkID: "workID1",
					},
					{
						WorkID: "workID1",
					},
				},
				{
					{
						WorkID: "workID1",
					},
					{
						WorkID: "workID1",
					},
				},
			},
			afterAdd: []types.CoordinatedBlockProposal{},
			deleteProposals: []types.CoordinatedBlockProposal{
				{
					WorkID: "workID1",
				},
			},
			afterDelete: []types.CoordinatedBlockProposal{},
			timeFn: func() time.Time {
				return time.Now().Add(-72 * time.Hour)
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			oldTimeFn := timeFn
			timeFn = tc.timeFn
			defer func() {
				timeFn = oldTimeFn
			}()

			store := NewMetadataStore(nil, nil)
			for _, proposal := range tc.addProposals {
				store.AddLogRecoveryProposal(proposal...)
			}
			proposals := store.ViewLogRecoveryProposal()
			assert.Equal(t, tc.afterAdd, proposals)
			store.RemoveLogRecoveryProposal(tc.deleteProposals...)
			proposals = store.ViewLogRecoveryProposal()
			assert.Equal(t, tc.afterDelete, proposals)
		})

	}
}

type mockBlockSubscriber struct {
	ch chan types.BlockHistory
}

func (_m *mockBlockSubscriber) Subscribe() (int, chan types.BlockHistory, error) {
	return 0, _m.ch, nil
}

func (_m *mockBlockSubscriber) Unsubscribe(int) error {
	return nil
}
