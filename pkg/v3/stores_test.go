package ocr2keepers

import (
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
