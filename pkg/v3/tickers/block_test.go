package tickers

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg"
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

func TestBlockTicker(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch := make(chan ocr2keepers.BlockHistory)

	sub := &mockSubscriber{
		SubscribeFn: func() (int, chan ocr2keepers.BlockHistory, error) {
			return 0, ch, nil
		},
		UnsubscribeFn: func(id int) error {
			return nil
		},
	}
	ticker, err := NewBlockTicker(sub)
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		err = ticker.Start(ctx)
		assert.NoError(t, err)
	}()

	firstBlockHistory := ocr2keepers.BlockHistory{ocr2keepers.BlockKey("key 1"), ocr2keepers.BlockKey("key 2")}
	secondBlockHistory := ocr2keepers.BlockHistory{ocr2keepers.BlockKey("key 3")}
	thirdBlockHistory := ocr2keepers.BlockHistory{ocr2keepers.BlockKey("key 4")}

	blockHistories := []ocr2keepers.BlockHistory{
		firstBlockHistory,
		secondBlockHistory,
		thirdBlockHistory,
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		if got := <-ticker.C; !reflect.DeepEqual(firstBlockHistory, got) {
			t.Errorf("expected %v, but got %v", firstBlockHistory, got)
		}
		wg.Done()
	}()

	time.Sleep(100 * time.Millisecond)

	for _, blockHistory := range blockHistories {
		ch <- blockHistory
	}

	wg.Wait()

	wg.Add(1)
	go func() {
		if got := <-ticker.C; !reflect.DeepEqual(thirdBlockHistory, got) {
			t.Errorf("expected %v, but got %v", thirdBlockHistory, got)
		}
		wg.Done()
	}()

	time.Sleep(100 * time.Millisecond)

	ch <- thirdBlockHistory

	wg.Wait()
	ticker.Close()

}

func TestBlockTicker_buffered(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch := make(chan ocr2keepers.BlockHistory)

	sub := &mockSubscriber{
		SubscribeFn: func() (int, chan ocr2keepers.BlockHistory, error) {
			return 0, ch, nil
		},
		UnsubscribeFn: func(id int) error {
			return nil
		},
	}
	ticker, err := NewBlockTicker(sub)
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		err = ticker.Start(ctx)
		assert.NoError(t, err)
	}()

	firstBlockHistory := ocr2keepers.BlockHistory{ocr2keepers.BlockKey("key 1"), ocr2keepers.BlockKey("key 2")}
	secondBlockHistory := ocr2keepers.BlockHistory{ocr2keepers.BlockKey("key 3")}
	thirdBlockHistory := ocr2keepers.BlockHistory{ocr2keepers.BlockKey("key 4")}

	blockHistories := []ocr2keepers.BlockHistory{
		firstBlockHistory,
		secondBlockHistory,
		thirdBlockHistory,
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		if got := <-ticker.C; !reflect.DeepEqual(firstBlockHistory, got) {
			t.Errorf("expected %v, but got %v", firstBlockHistory, got)
		}
		time.Sleep(100 * time.Millisecond)
		// the third block history should be the last one received, and made availble on t.bufferedValue to be sent when t.C becomes available
		if got := <-ticker.C; !reflect.DeepEqual(thirdBlockHistory, got) {
			t.Errorf("expected %v, but got %v", thirdBlockHistory, got)
		}
		wg.Done()
	}()

	time.Sleep(100 * time.Millisecond)

	for _, blockHistory := range blockHistories {
		ch <- blockHistory
	}

	wg.Wait()

	ticker.Close()

}

func TestBlockTicker_cancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	ch := make(chan ocr2keepers.BlockHistory, 1)

	sub := &mockSubscriber{
		SubscribeFn: func() (int, chan ocr2keepers.BlockHistory, error) {
			return 0, ch, nil
		},
		UnsubscribeFn: func(id int) error {
			return nil
		},
	}

	ticker, err := NewBlockTicker(sub)
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		err = ticker.Start(ctx)
		assert.Error(t, err) // context canceled
	}()

	cancel()
}

func TestBlockTicker_subscribeError(t *testing.T) {
	sub := &mockSubscriber{
		SubscribeFn: func() (int, chan ocr2keepers.BlockHistory, error) {
			return 0, nil, errors.New("subscribe failure")
		},
		UnsubscribeFn: func(id int) error {
			return nil
		},
	}

	_, err := NewBlockTicker(sub)
	assert.Error(t, err)
}

func TestBlockTicker_unsubscribeError(t *testing.T) {
	ch := make(chan ocr2keepers.BlockHistory, 1)

	sub := &mockSubscriber{
		SubscribeFn: func() (int, chan ocr2keepers.BlockHistory, error) {
			return 0, ch, nil
		},
		UnsubscribeFn: func(id int) error {
			return errors.New("unsubscribe failure")
		},
	}

	ticker, err := NewBlockTicker(sub)
	assert.Nil(t, err)

	go func() {
		err = ticker.Start(context.Background())
		assert.NoError(t, err) // context canceled
	}()

	ticker.Close()
}
