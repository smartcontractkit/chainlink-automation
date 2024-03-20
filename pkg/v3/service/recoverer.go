package service

import (
	"context"
	"log"
	"runtime/debug"
	"time"
)

const (
	panicRestartWait = 10 * time.Second
)

// Recoverable is a service that a Recoverer can manage
type Recoverable interface {
	// Start is a function that is expected to block and only return on
	// completion either with an error or nil
	Start(context.Context) error
	// Close stops the execution of the blocking Start function and causes an
	// immediate return
	Close() error
}

type serviceRecoverer struct {
	recoverable Recoverable
	errCh       chan error // Channel to send e
	quitCh      chan struct{}
	finishedCh  chan struct{}
	log         *log.Logger
	coolDown    time.Duration
}

func NewServiceRecoverer(recoverable Recoverable, log *log.Logger) *serviceRecoverer {
	return &serviceRecoverer{
		recoverable: recoverable,
		errCh:       make(chan error, 1),
		quitCh:      make(chan struct{}, 1),
		finishedCh:  make(chan struct{}, 1),
		coolDown:    panicRestartWait,
		log:         log,
	}
}

func (r *serviceRecoverer) Start(ctx context.Context) error {
	go func() {
		for {
			shouldRecover := false
			select {
			case <-r.quitCh:
				r.log.Println("stopping recoverer...")
				return
			case <-r.finishedCh:
				r.log.Println("finished recoverable service, stopping recoverer...")
				return
			default:
				func() {
					defer func() {
						if err := recover(); err != nil {
							if r.log != nil {
								r.log.Println(err)
								r.log.Println(string(debug.Stack()))
							}
							shouldRecover = true
						}
					}()
					if err := r.recoverable.Start(ctx); err != nil {
						r.log.Printf("error encountered running service: %s", err.Error())
						shouldRecover = true
					}
				}()
				if shouldRecover {
					<-time.After(r.coolDown)
				} else {
					close(r.finishedCh)
				}
			}
		}
	}()
	return nil
}

func (r *serviceRecoverer) Close() error {
	r.recoverable.Close()
	close(r.quitCh)
	return nil
}
