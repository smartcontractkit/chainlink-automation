package net

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"runtime"
	"sync"
	"time"

	"gonum.org/v1/gonum/stat/distuv"

	"github.com/smartcontractkit/ocr2keepers/cmd/simv3/util"
)

const (
	RateLimitTimeout    = 50 * time.Millisecond
	ProbabilityAccuracy = 10000
)

var (
	ErrNetRateLimitExceeded = fmt.Errorf("network service rate limit exceeded")
	ErrNetLoadLimitExceeded = fmt.Errorf("network service load limit exceeded")
)

type NetTelemetry interface {
	Register(string, time.Duration, error)
	AddRateDataPoint(int)
}

type Generator interface {
	Rand() float64
}

type SimulatedNetworkService struct {
	LoadLimitProbability float64
	RateLimit            int
	AvgLatency           int
	Name                 string

	telemetry    NetTelemetry
	distribution Generator
	done         chan struct{}

	mu                 sync.RWMutex
	totalCalls         int
	callCountIncrement int
	dataPoints         []int
}

// NewSimulatedNetworkService creates a network service with provided load,
// rate, and latency configurations and pre-configured binomial distribution for
// applying network wait times. The service starts after initialization and
// stops using a runtime finalizer.
func NewSimulatedNetworkService(load float64, rate, latency int, tel NetTelemetry) *SimulatedNetworkService {
	if load > 1 || load < 0 {
		panic("load limit probability must be between 0 and 1")
	}

	if tel == nil {
		panic("unexpected nil telemetry collector")
	}

	rpc := &SimulatedNetworkService{
		LoadLimitProbability: load,
		RateLimit:            rate,
		AvgLatency:           latency,
		distribution: distuv.Binomial{
			N:   float64(latency * 2),
			P:   0.4,
			Src: util.NewCryptoRandSource(),
		},
		dataPoints: []int{},
		telemetry:  tel,
		done:       make(chan struct{}),
	}

	go rpc.run()

	runtime.SetFinalizer(rpc, func(srv *SimulatedNetworkService) { srv.stop() })

	return rpc
}

func (sim *SimulatedNetworkService) Call(ctx context.Context, name string) <-chan error {
	sim.addCall()

	chOut := make(chan error, 1)

	// shortcut if rate limited
	if sim.shouldRateLimit() {
		go sim.doErroredLimit(ctx, name, ErrNetRateLimitExceeded, chOut)
	} else if sim.atLoadLimit() {
		go sim.doErroredLimit(ctx, name, ErrNetLoadLimitExceeded, chOut)
	} else {
		go sim.doForcedWait(ctx, name, chOut)
	}

	return chOut
}

func (sim *SimulatedNetworkService) addCall() {
	sim.mu.Lock()
	defer sim.mu.Unlock()

	sim.totalCalls++
	sim.callCountIncrement++
}

func (sim *SimulatedNetworkService) shouldRateLimit() bool {
	return sim.rate() > sim.RateLimit
}

func (sim *SimulatedNetworkService) atLoadLimit() bool {
	accuracy := ProbabilityAccuracy
	randValue := rand.Intn(accuracy)
	probabilityValue := int(math.Round(sim.LoadLimitProbability * float64(accuracy)))

	return randValue < probabilityValue
}

func (sim *SimulatedNetworkService) rate() int {
	sim.mu.RLock()
	defer sim.mu.RUnlock()

	// total of last 10 data points must be more than configured limit
	dataPointLimit := len(sim.dataPoints)
	if dataPointLimit > 10 {
		dataPointLimit = 10
	}

	subset := sim.dataPoints[len(sim.dataPoints)-dataPointLimit:]

	calls := 0
	for _, val := range subset {
		calls += val
	}

	return calls
}

func (sim *SimulatedNetworkService) doErroredLimit(ctx context.Context, name string, err error, chErr chan error) {
	err = fmt.Errorf("%s %w", sim.Name, err)
	timer := time.NewTimer(RateLimitTimeout)

	sim.telemetry.Register(name, RateLimitTimeout, err)

	// pass the error after either the context times out or after the timer
	select {
	case <-ctx.Done():
		timer.Stop()

		chErr <- ctx.Err()
	case <-timer.C:
		chErr <- err
	}
}

func (sim *SimulatedNetworkService) doForcedWait(ctx context.Context, name string, chErr chan error) {
	randVal := sim.distribution.Rand()
	waitTime := (time.Duration(randVal) * time.Millisecond) + (RateLimitTimeout)
	timer := time.NewTimer(waitTime)

	select {
	case <-ctx.Done():
		sim.telemetry.Register(name, waitTime, ctx.Err())

		timer.Stop()

		chErr <- ctx.Err()
	case <-timer.C:
		sim.telemetry.Register(name, waitTime, nil)

		chErr <- nil
	}
}

func (sim *SimulatedNetworkService) collectIncrement() int {
	sim.mu.Lock()
	defer sim.mu.Unlock()

	calls := sim.callCountIncrement

	sim.callCountIncrement = 0
	sim.dataPoints = append(sim.dataPoints, calls)

	sim.telemetry.AddRateDataPoint(calls)

	return calls
}

func (sim *SimulatedNetworkService) run() {
	ticker := time.NewTicker(100 * time.Millisecond)

	for {
		select {
		case <-ticker.C:
			sim.collectIncrement()
		case <-sim.done:
			ticker.Stop()
			return
		}
	}
}

func (sim *SimulatedNetworkService) stop() {
	close(sim.done)
}
