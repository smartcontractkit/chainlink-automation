package util

import (
	"context"
	"errors"
	"fmt"
	"log"
	"runtime/debug"
	"sync"
	"time"
)

var (
	errServiceStopped = fmt.Errorf("service stopped")
	coolDown          = 10 * time.Second
)

type Doable interface {
	Do() error
	Stop()
}

func NewRecoverableService(svc Doable, logger *log.Logger) *RecoverableService {
	ctx, cancel := context.WithCancel(context.Background())
	return &RecoverableService{
		service: svc,
		stopped: make(chan error, 1),
		log:     logger,
		ctx:     ctx,
		cancel:  cancel,
	}
}

type RecoverableService struct {
	mu      sync.Mutex
	running bool
	service Doable
	stopped chan error
	log     *log.Logger
	ctx     context.Context
	cancel  context.CancelFunc
}

func (m *RecoverableService) Start() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		return
	}

	go m.serviceStart()
	m.running = true
}

func (m *RecoverableService) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running {
		return
	}

	m.service.Stop()
	m.cancel()
	m.running = false
}

func (m *RecoverableService) serviceStart() {
	m.run()
	for {
		select {
		case err := <-m.stopped:
			// restart the service
			if err != nil && errors.Is(err, errServiceStopped) {
				<-time.After(coolDown)
				m.run()
			}
		case <-m.ctx.Done():
			return
		}
	}
}

func (m *RecoverableService) run() {
	go func(s Doable, l *log.Logger, chStop chan error, ctx context.Context) {
		defer func() {
			if err := recover(); err != nil {
				if l != nil {
					l.Println(err)
					l.Println(string(debug.Stack()))
				}

				chStop <- errServiceStopped
			}
		}()

		err := s.Do()

		if l != nil {
			l.Println(err)
		}

		chStop <- err
	}(m.service, m.log, m.stopped, m.ctx)
}
