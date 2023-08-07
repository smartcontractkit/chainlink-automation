package store

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/smartcontractkit/ocr2keepers/pkg/v3/tickers"
	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg/v3/types"

	"github.com/stretchr/testify/assert"
)

func TestMetadataSetGetDelete(t *testing.T) {
	md := NewMetadata(nil)

	var key MetadataKey = "test key"

	value := "test value"
	valueUpdate := "test value 2"

	if _, ok := md.Get(key); ok {
		t.Log("key should not have a value")
		t.FailNow()
	}

	md.Set(key, value)

	v, ok := md.Get(key)
	if !ok {
		t.Log("key should have a value")
		t.FailNow()
	}

	assert.Equal(t, value, v, "value should be %s", value)

	md.Set(key, valueUpdate)

	v, ok = md.Get(key)
	if !ok {
		t.Log("key should have a value")
		t.FailNow()
	}

	assert.Equal(t, valueUpdate, v, "value should be reset to %s", valueUpdate)

	md.Delete(key)
	if _, ok := md.Get(key); ok {
		t.Log("key should not have a value")
		t.FailNow()
	}
}

func TestBlockSource(t *testing.T) {
	ch := make(chan ocr2keepers.BlockHistory)
	bs := &mockSubscriber{
		SubscribeFn: func() (int, chan ocr2keepers.BlockHistory, error) {
			return 0, ch, nil
		},
		UnsubscribeFn: func(id int) error {
			return nil
		},
	}

	ticker, err := tickers.NewBlockTicker(bs)
	assert.NoError(t, err, "no error on ticker create")

	str := NewMetadata(ticker)

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		assert.NoError(t, ticker.Start(context.Background()))
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		assert.NoError(t, str.Start(context.Background()))
		wg.Done()
	}()

	historyData := ocr2keepers.BlockHistory{
		("3"),
		("2"),
		("1"),
	}

	ch <- historyData

	time.Sleep(20 * time.Millisecond)

	history, ok := str.Get(BlockHistoryMetadata)
	assert.True(t, ok, "block history data should exist")
	assert.Equal(t, historyData, history, "block history value should match input")
	assert.NoError(t, ticker.Close(), "no error on close")
	assert.NoError(t, str.Close(), "no error on close")

	wg.Wait()
}

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
