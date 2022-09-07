package main

import (
	"context"
	"log"
	"sync"
	"time"
)

type OCRCallBase struct {
	Context context.Context
	Round   int
	Epoch   int
}

type OCRCall[T OCRQuery | []OCRObservation | OCRReport | struct{}] struct {
	OCRCallBase
	Data T
}

type OCRQuery []byte
type OCRObservation []byte
type OCRReport []byte

type OCRController struct {
	RoundTime       time.Duration
	Queries         chan OCRQuery
	Observations    chan OCRObservation
	Reports         chan OCRReport
	Stop            chan struct{}
	Receivers       []*OCRReceiver
	MaxRounds       int
	mu              sync.Mutex
	collection      []OCRObservation
	completeReports [][]OCRReport
}

func NewOCRController(round time.Duration, rnds int, rcvrs ...*OCRReceiver) *OCRController {
	return &OCRController{
		RoundTime:       round,
		Queries:         make(chan OCRQuery, 1),
		Observations:    make(chan OCRObservation, len(rcvrs)),
		Reports:         make(chan OCRReport, len(rcvrs)),
		Stop:            make(chan struct{}, 1),
		Receivers:       rcvrs,
		MaxRounds:       rnds,
		collection:      []OCRObservation{},
		completeReports: [][]OCRReport{},
	}
}

func (c *OCRController) sendInitCall(ctx context.Context, call OCRCallBase) {
	var wg sync.WaitGroup

	// TODO: pick randomly from available recievers
	// send init call to single node in receivers
	// mimicks leader selection in OCR
	wg.Add(1)

	go func(r *OCRReceiver) {
		defer wg.Done()

		select {
		case r.Init <- OCRCall[struct{}]{OCRCallBase: call, Data: struct{}{}}:
			log.Printf("controller: init call sent to %s", r.Name)
			return
		case <-ctx.Done():
			return
		}
	}(c.Receivers[0])

	wg.Wait()
}

func (c *OCRController) sendQueries(ctx context.Context, call OCRCallBase) {
	var wg sync.WaitGroup

	select {
	case q := <-c.Queries:
		log.Printf("controller: received query from leader")
		for _, rec := range c.Receivers {
			wg.Add(1)

			go func(r *OCRReceiver) {
				defer wg.Done()

				select {
				case r.Query <- OCRCall[OCRQuery]{OCRCallBase: call, Data: q}:
					log.Printf("controller: sent query to %s", r.Name)
					return
				case <-ctx.Done():
					return
				}
			}(rec)
		}

		wg.Wait()
	case <-ctx.Done():
		return
	}
}

func (c *OCRController) collectObservations(ctx context.Context) {
	var wg sync.WaitGroup

	c.mu.Lock()
	c.collection = make([]OCRObservation, len(c.Receivers))
	c.mu.Unlock()

	for j := 0; j < len(c.Receivers); j++ {
		wg.Add(1)

		go func(idx int) {
			defer wg.Done()

			select {
			case o := <-c.Observations:
				log.Printf("controller: received observation from %s", c.Receivers[idx].Name)
				c.mu.Lock()
				c.collection[idx] = o
				c.mu.Unlock()
				return
			case <-ctx.Done():
				return
			}
		}(j)
	}

	wg.Wait()
}

func (c *OCRController) sendObservations(ctx context.Context, call OCRCallBase) {
	var wg sync.WaitGroup

	// send observations to all nodes
	c.mu.Lock()
	copy := c.collection
	c.mu.Unlock()

	for _, rec := range c.Receivers {
		wg.Add(1)

		go func(r *OCRReceiver) {
			defer wg.Done()

			select {
			case r.Observations <- OCRCall[[]OCRObservation]{OCRCallBase: call, Data: copy}:
				log.Printf("controller: sent observations to %s", r.Name)
				return
			case <-ctx.Done():
				return
			}
		}(rec)
	}

	wg.Wait()
}

func (c *OCRController) collectReports(ctx context.Context, call OCRCallBase) {
	var wg sync.WaitGroup

	stp, isSet := ctx.Deadline()
	if !isSet {
		stp = time.Now()
	}

	stp = stp.Add(-1 * time.Duration(len(c.Receivers)) * 20 * time.Millisecond)

	if stp.Before(time.Now()) {
		return
	}

	wg.Add(1)
	go func() {
		defer wg.Done()

		rpts := []OCRReport{}

	Outer:
		for i := 0; i < len(c.Receivers); i++ {
			select {
			case r := <-c.Reports:
				log.Printf("controller: report received from %s", c.Receivers[i].Name)
				rpts = append(rpts, r)
			case <-ctx.Done():
				break Outer
			}
		}

		c.mu.Lock()
		c.completeReports = append(c.completeReports, rpts)
		c.mu.Unlock()
	}()
	wg.Wait()

	c.mu.Lock()
	if len(c.completeReports[len(c.completeReports)-1]) > 0 {
		rpt := c.collapseReports(c.completeReports[len(c.completeReports)-1])

		for _, rec := range c.Receivers {
			wg.Add(1)
			go func(r *OCRReceiver) {
				defer wg.Done()

				select {
				case r.Report <- OCRCall[OCRReport]{OCRCallBase: call, Data: rpt}:
					log.Printf("controller: sent final report to %s", r.Name)
					return
				case <-ctx.Done():
					return
				}
			}(rec)
		}
	}
	c.mu.Unlock()

	wg.Wait()
}

func (c *OCRController) collapseReports(reports []OCRReport) OCRReport {
	// TODO: nieve implementation assumes are reports are the same and sends the first
	return reports[0]
}

func (c *OCRController) stopReceivers() {
	for _, rec := range c.Receivers {
		go func(r *OCRReceiver) {
			r.Stop <- struct{}{}
		}(rec)
	}
}

func (c *OCRController) Start(ctx context.Context) chan struct{} {
	done := make(chan struct{}, 1)

	go func() {
		//tkr := time.NewTicker(c.RoundTime)
		tkr := 0 * time.Second
		iterations := 0

		log.Printf("starting OCR controller with round time of %d seconds", c.RoundTime/time.Second)

		for {
			t := time.NewTimer(tkr)

			select {
			case <-c.Stop:
				log.Printf("receivers stopping")
				c.stopReceivers()
				log.Printf("receivers stopped")
				t.Stop()
				done <- struct{}{}
				return
			case <-t.C:
				iterations++
				roundCtx, cancel := context.WithDeadline(ctx, time.Now().Add(c.RoundTime-(10*time.Millisecond)))
				base := OCRCallBase{Context: roundCtx, Round: iterations, Epoch: 1}

				log.Printf("-----> round %d begins", iterations)

				// send call to Query to begin round
				c.sendInitCall(roundCtx, base)

				// send Query to all recievers to begin creating Observations
				c.sendQueries(roundCtx, base)

				// collect all observations from receivers
				c.collectObservations(roundCtx)

				// send observations to all receivers
				c.sendObservations(roundCtx, base)

				c.collectReports(roundCtx, base)

				log.Printf("<----- round %d ends", iterations)
				if c.MaxRounds > 0 && iterations == c.MaxRounds {
					log.Printf("max rounds encountered; terminating process")
					c.Stop <- struct{}{}
				}
				tkr = c.RoundTime
				cancel()
			case <-ctx.Done():
				// stop all receivers
				c.Stop <- struct{}{}
			}
		}
	}()

	return done
}
