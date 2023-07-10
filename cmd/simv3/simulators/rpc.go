package simulators

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"runtime"
	"sync"
	"time"

	"gonum.org/v1/gonum/stat/distuv"
)

var (
	ErrRPCContextCancelled  = fmt.Errorf("rpc context cancelled")
	ErrRPCRateLimitExceeded = fmt.Errorf("rpc rate limit exceeded")
	ErrRPCLoadLimitExceeded = fmt.Errorf("rpc load limit exceeded")
)

type Generator interface {
	Rand() float64
}

type SimulatedRPC struct {
	LoadLimitProbability float64
	RateLimit            int
	AvgLatency           int
	data                 RPCTelemetry
	mu                   sync.Mutex
	distribution         Generator
	totalCalls           int
	callCountIncrement   int
	dataPoints           []int
	done                 chan struct{}
}

func NewSimulatedRPC(load float64, rate, latency int, tel RPCTelemetry) *SimulatedRPC {
	if load > 1 || load < 0 {
		panic("load limit probability must be between 0 and 1")
	}

	rpc := &SimulatedRPC{
		LoadLimitProbability: load,
		RateLimit:            rate,
		AvgLatency:           latency,
		distribution: distuv.Binomial{
			N:   float64(latency * 2),
			P:   0.4,
			Src: newCryptoRandSource(),
		},
		dataPoints: []int{},
		data:       tel,
		done:       make(chan struct{}),
	}

	go rpc.run()

	runtime.SetFinalizer(rpc, func(srv *SimulatedRPC) { srv.stop() })

	return rpc
}

func (rpc *SimulatedRPC) Call(ctx context.Context, name string) <-chan error {
	rpc.mu.Lock()
	defer rpc.mu.Unlock()

	rpc.totalCalls++
	rpc.callCountIncrement++

	chOut := make(chan error, 1)

	// shortcut if rate limited
	if rpc.shouldRateLimit() {
		go func(ch chan error, tel RPCTelemetry, n string) {
			l := 50 * time.Millisecond
			err := ErrRPCRateLimitExceeded

			if tel != nil {
				tel.RegisterCall(n, l, err)
			}

			<-time.After(l)
			ch <- err
		}(chOut, rpc.data, name)
	} else if rpc.atLoadLimit() {
		go func(ch chan error, tel RPCTelemetry, n string) {
			l := 50 * time.Millisecond
			err := ErrRPCLoadLimitExceeded

			if tel != nil {
				tel.RegisterCall(n, l, err)
			}

			<-time.After(l)
			ch <- err
		}(chOut, rpc.data, name)
	} else {
		r := rpc.distribution.Rand()
		lat := (time.Duration(r) * time.Millisecond) + (50 * time.Millisecond)

		go func(ch chan error, l time.Duration, tel RPCTelemetry, n string) {
			t := time.NewTimer(l)

			select {
			case <-ctx.Done():
				if tel != nil {
					tel.RegisterCall(n, l, ErrRPCContextCancelled)
				}

				t.Stop()
				ch <- ErrRPCContextCancelled
			case <-t.C:
				if tel != nil {
					tel.RegisterCall(n, l, nil)
				}
				ch <- nil
			}
		}(chOut, lat, rpc.data, name)
	}

	return chOut
}

func (rpc *SimulatedRPC) rate() int {
	// total of last 10 data points must be more than configured limit
	l := len(rpc.dataPoints)
	if l > 10 {
		l = 10
	}

	subset := rpc.dataPoints[len(rpc.dataPoints)-l:]

	calls := 0
	for _, val := range subset {
		calls += val
	}

	return calls
}

func (rpc *SimulatedRPC) shouldRateLimit() bool {
	return rpc.rate() > rpc.RateLimit
}

func (rpc *SimulatedRPC) atLoadLimit() bool {
	accuracy := 10000
	r := rand.Intn(accuracy)
	v := rpc.LoadLimitProbability * float64(accuracy)
	return r < int(math.Round(v))
}

func (rpc *SimulatedRPC) collectIncrement() int {
	rpc.mu.Lock()
	defer rpc.mu.Unlock()

	calls := rpc.callCountIncrement
	rpc.callCountIncrement = 0
	rpc.dataPoints = append(rpc.dataPoints, calls)

	if rpc.data != nil {
		rpc.data.AddRateDataPoint(calls)
	}

	return calls
}

func (rpc *SimulatedRPC) run() {
	ticker := time.NewTicker(100 * time.Millisecond)

	for {
		select {
		case <-ticker.C:
			rpc.collectIncrement()
		case <-rpc.done:
			ticker.Stop()
			return
		}
	}
}

func (rpc *SimulatedRPC) stop() {
	close(rpc.done)
}
