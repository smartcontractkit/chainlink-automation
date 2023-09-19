package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

func contextWithInterrupt(ctx context.Context) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(ctx)

	go func() {
		chSigTerm := make(chan os.Signal, 1)

		signal.Notify(chSigTerm, os.Interrupt, syscall.SIGTERM)

	ServiceLoop:
		for {
			select {
			case <-chSigTerm:
				cancel()

				break ServiceLoop
			case <-ctx.Done():
				break ServiceLoop
			}
		}
	}()

	return ctx, cancel
}
