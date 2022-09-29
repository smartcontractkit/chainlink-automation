package ocr2keepers

import "github.com/smartcontractkit/ocr2keepers/pkg/types"

type NotifierRetriever[T any] interface {
	Notify() chan struct{}
	Retrieve() (T, bool)
}

type WrappedPerformLogProvider struct {
	src  NotifierRetriever[types.PerformLog]
	stop chan struct{}
	msgs chan types.PerformLog
}

func NewWrappedPerformLogProvider(nr NotifierRetriever[types.PerformLog]) *WrappedPerformLogProvider {
	p := &WrappedPerformLogProvider{
		src:  nr,
		stop: make(chan struct{}, 10),
		msgs: make(chan types.PerformLog, 1),
	}

	go func() {
		defer func() {
			if err := recover(); err != nil {
				p.stop <- struct{}{}
			}
		}()

		for {
			select {
			case <-p.src.Notify():
				msg, ok := p.src.Retrieve()
				if ok {
					p.msgs <- msg
				}
			case <-p.stop:
				return
			}
		}
	}()

	return p
}

func (p *WrappedPerformLogProvider) Subscribe() chan types.PerformLog {
	return p.msgs
}

func (p *WrappedPerformLogProvider) Unsubscribe() {
	p.stop <- struct{}{}
}
