package stores

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/smartcontractkit/chainlink-automation/pkg/v3/types"
	commontypes "github.com/smartcontractkit/chainlink-common/pkg/types/automation"
)

func TestNewMetadataStore(t *testing.T) {
	t.Run("creating the metadata store errors when subscribing to the block subscriber errors", func(t *testing.T) {
		blockSubscriber := &mockBlockSubscriber{
			SubscribeFn: func() (int, chan commontypes.BlockHistory, error) {
				return 0, nil, errors.New("subscribe boom")
			},
		}

		_, err := NewMetadataStore(blockSubscriber, nil)
		assert.Error(t, err)
		assert.Equal(t, "subscribe boom", err.Error())

	})

	t.Run("the metadata store starts, reads from the block ticker, and stops without erroring", func(t *testing.T) {
		canClose := make(chan struct{}, 1)
		finished := make(chan struct{}, 1)
		ch := make(chan commontypes.BlockHistory)
		blockSubscriber := &mockBlockSubscriber{
			SubscribeFn: func() (int, chan commontypes.BlockHistory, error) {
				return 1, ch, nil
			},
			UnsubscribeFn: func(i int) error {
				return nil
			},
		}

		go func() {
			ch <- commontypes.BlockHistory{
				commontypes.BlockKey{
					Number: 4,
				},
				commontypes.BlockKey{
					Number: 3,
				},
			}
		}()

		store, err := NewMetadataStore(blockSubscriber, nil)
		assert.NoError(t, err)

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

	t.Run("closing the metadata store errors when unsubscribing from the block subscriber errors", func(t *testing.T) {
		ch := make(chan commontypes.BlockHistory)
		blockSubscriber := &mockBlockSubscriber{
			SubscribeFn: func() (int, chan commontypes.BlockHistory, error) {
				return 1, ch, nil
			},
			UnsubscribeFn: func(i int) error {
				return errors.New("unsubscribe boom")
			},
		}

		store, err := NewMetadataStore(blockSubscriber, nil)
		assert.NoError(t, err)

		store.running.Store(true)

		closeErr := store.Close()
		assert.Error(t, closeErr)
		assert.Equal(t, "unsubscribe boom", closeErr.Error())

	})

	t.Run("the metadata store starts, reads from the block ticker, and stops via a cancelled context without erroring", func(t *testing.T) {
		canClose := make(chan struct{}, 1)
		finished := make(chan struct{}, 1)
		ch := make(chan commontypes.BlockHistory)
		blockSubscriber := &mockBlockSubscriber{
			SubscribeFn: func() (int, chan commontypes.BlockHistory, error) {
				return 1, ch, nil
			},
			UnsubscribeFn: func(i int) error {
				return nil
			},
		}

		go func() {
			ch <- commontypes.BlockHistory{
				commontypes.BlockKey{
					Number: 4,
				},
				commontypes.BlockKey{
					Number: 3,
				},
			}
		}()

		store, err := NewMetadataStore(blockSubscriber, nil)
		assert.NoError(t, err)

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
		ch := make(chan commontypes.BlockHistory)
		blockSubscriber := &mockBlockSubscriber{
			SubscribeFn: func() (int, chan commontypes.BlockHistory, error) {
				return 1, ch, nil
			},
		}
		store, _ := NewMetadataStore(blockSubscriber, nil)
		store.running.Store(true)
		err := store.Start(context.Background())
		assert.Error(t, err)
	})

	t.Run("closing an already closed metadata store returns an error", func(t *testing.T) {
		ch := make(chan commontypes.BlockHistory)
		blockSubscriber := &mockBlockSubscriber{
			SubscribeFn: func() (int, chan commontypes.BlockHistory, error) {
				return 1, ch, nil
			},
		}
		store, _ := NewMetadataStore(blockSubscriber, nil)
		store.running.Store(false)
		err := store.Close()
		assert.Error(t, err)
	})
}

func TestMetadataStore_AddConditionalProposal(t *testing.T) {
	for _, tc := range []struct {
		name            string
		addProposals    [][]commontypes.CoordinatedBlockProposal
		afterAdd        []commontypes.CoordinatedBlockProposal
		deleteProposals []commontypes.CoordinatedBlockProposal
		afterDelete     []commontypes.CoordinatedBlockProposal
		timeFn          func() time.Time
	}{
		{
			name: "all unique proposals are added and retrieved, existent keys are successfully deleted",
			addProposals: [][]commontypes.CoordinatedBlockProposal{
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
			afterAdd: []commontypes.CoordinatedBlockProposal{
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
			deleteProposals: []commontypes.CoordinatedBlockProposal{
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
			afterDelete: []commontypes.CoordinatedBlockProposal{
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
			addProposals: [][]commontypes.CoordinatedBlockProposal{
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
			afterAdd: []commontypes.CoordinatedBlockProposal{
				{
					WorkID: "workID1",
				},
			},
			deleteProposals: []commontypes.CoordinatedBlockProposal{
				{
					WorkID: "workID1",
				},
			},
			afterDelete: []commontypes.CoordinatedBlockProposal{},
			timeFn:      time.Now,
		},
		{
			name: "proposals added three days ago aren't returned, non existent keys result in a no op delete",
			addProposals: [][]commontypes.CoordinatedBlockProposal{
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
			afterAdd: []commontypes.CoordinatedBlockProposal{},
			deleteProposals: []commontypes.CoordinatedBlockProposal{
				{
					WorkID: "workID1",
				},
			},
			afterDelete: []commontypes.CoordinatedBlockProposal{},
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

			ch := make(chan commontypes.BlockHistory)
			blockSubscriber := &mockBlockSubscriber{
				SubscribeFn: func() (int, chan commontypes.BlockHistory, error) {
					return 1, ch, nil
				},
			}

			store, err := NewMetadataStore(blockSubscriber, nil)
			assert.NoError(t, err)

			for _, proposal := range tc.addProposals {
				store.addConditionalProposal(proposal...)
			}
			proposals := store.viewConditionalProposal()
			assert.Equal(t, tc.afterAdd, proposals)
			store.removeConditionalProposal(tc.deleteProposals...)
			proposals = store.viewConditionalProposal()
			assert.Equal(t, tc.afterDelete, proposals)
		})

	}
}

func TestMetadataStore_AddLogRecoveryProposal(t *testing.T) {
	for _, tc := range []struct {
		name            string
		addProposals    [][]commontypes.CoordinatedBlockProposal
		afterAdd        []commontypes.CoordinatedBlockProposal
		deleteProposals []commontypes.CoordinatedBlockProposal
		afterDelete     []commontypes.CoordinatedBlockProposal
		timeFn          func() time.Time
		typeGetter      types.UpkeepTypeGetter
	}{
		{
			name: "all unique proposals are added and retrieved, existent keys are successfully deleted",
			typeGetter: func(identifier commontypes.UpkeepIdentifier) types.UpkeepType {
				return types.LogTrigger
			},
			addProposals: [][]commontypes.CoordinatedBlockProposal{
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
			afterAdd: []commontypes.CoordinatedBlockProposal{
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
			deleteProposals: []commontypes.CoordinatedBlockProposal{
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
			afterDelete: []commontypes.CoordinatedBlockProposal{
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
			typeGetter: func(identifier commontypes.UpkeepIdentifier) types.UpkeepType {
				return types.LogTrigger
			},
			addProposals: [][]commontypes.CoordinatedBlockProposal{
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
			afterAdd: []commontypes.CoordinatedBlockProposal{
				{
					WorkID: "workID1",
				},
			},
			deleteProposals: []commontypes.CoordinatedBlockProposal{
				{
					WorkID: "workID1",
				},
			},
			afterDelete: []commontypes.CoordinatedBlockProposal{},
			timeFn:      time.Now,
		},
		{
			name: "proposals added three days ago aren't returned, non existent keys result in a no op delete",
			typeGetter: func(identifier commontypes.UpkeepIdentifier) types.UpkeepType {
				return types.LogTrigger
			},
			addProposals: [][]commontypes.CoordinatedBlockProposal{
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
			afterAdd: []commontypes.CoordinatedBlockProposal{},
			deleteProposals: []commontypes.CoordinatedBlockProposal{
				{
					WorkID: "workID1",
				},
			},
			afterDelete: []commontypes.CoordinatedBlockProposal{},
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
			ch := make(chan commontypes.BlockHistory)
			blockSubscriber := &mockBlockSubscriber{
				SubscribeFn: func() (int, chan commontypes.BlockHistory, error) {
					return 1, ch, nil
				},
			}

			store, err := NewMetadataStore(blockSubscriber, tc.typeGetter)
			assert.NoError(t, err)

			for _, proposal := range tc.addProposals {
				store.AddProposals(proposal...)
			}
			proposals := store.viewLogRecoveryProposal()
			assert.Equal(t, tc.afterAdd, proposals)
			store.RemoveProposals(tc.deleteProposals...)
			proposals = store.viewLogRecoveryProposal()
			assert.Equal(t, tc.afterDelete, proposals)
		})

	}
}

type mockBlockSubscriber struct {
	SubscribeFn   func() (int, chan commontypes.BlockHistory, error)
	UnsubscribeFn func(int) error
	StartFn       func(ctx context.Context) error
	CloseFn       func() error
}

func (_m *mockBlockSubscriber) Subscribe() (int, chan commontypes.BlockHistory, error) {
	return _m.SubscribeFn()
}

func (_m *mockBlockSubscriber) Unsubscribe(i int) error {
	return _m.UnsubscribeFn(i)
}
func (r *mockBlockSubscriber) Start(ctx context.Context) error {
	return r.StartFn(ctx)
}
func (r *mockBlockSubscriber) Close() error {
	return r.CloseFn()
}
