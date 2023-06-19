package ocr2keepers

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/tickers"
)

type mockSubscriber struct {
	SubscribeFn   func() (int, chan ocr2keepers.BlockHistory, error)
	UnsubscribeFn func(id int) error
}

func (s *mockSubscriber) Subscribe() (int, chan ocr2keepers.BlockHistory, error) {
	return s.SubscribeFn()
}

func (s *mockSubscriber) Unsubscribe(id int) error {
	return s.UnsubscribeFn(id)
}

func TestNewMetadataStore(t *testing.T) {
	t.Run("sets the incoming block histories", func(t *testing.T) {
		ch := make(chan ocr2keepers.BlockHistory)

		subscriber := &mockSubscriber{
			SubscribeFn: func() (int, chan ocr2keepers.BlockHistory, error) {
				return 0, ch, nil
			},
			UnsubscribeFn: func(id int) error {
				return nil
			},
		}

		ticker, err := tickers.NewBlockTicker(subscriber)
		assert.NoError(t, err)
		store := NewMetadataStore(ticker)

		go func() {
			err := ticker.Start(context.Background())
			assert.NoError(t, err)
		}()

		go func() {
			err := store.Start()
			assert.NoError(t, err)
		}()

		identifiers := []ocr2keepers.UpkeepIdentifier{
			[]byte("12|34"),
			[]byte("56|78"),
		}

		err = store.Set(identifiers)
		assert.NoError(t, err)

		assert.True(t, reflect.DeepEqual(store.identifiers, identifiers))

		history := ocr2keepers.BlockHistory{
			ocr2keepers.BlockKey("key1"),
		}

		ch <- history

		assert.Eventually(t, func() bool {
			store.m.RLock()
			defer store.m.RUnlock()
			return reflect.DeepEqual(store.blockHistory, store.blockHistory)
		}, 10*time.Second, time.Second)

		err = store.Stop()
		assert.NoError(t, err)

	})
}
