package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"runtime/debug"
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
	RoundTime            time.Duration
	QueryTime            time.Duration
	ObservationTime      time.Duration
	ReportTime           time.Duration
	Queries              chan OCRQuery
	Observations         chan OCRObservation
	Reports              chan OCRReport
	Stop                 chan struct{}
	Receivers            []*OCRReceiver
	MaxRounds            int
	MaxQueryLength       int
	MaxObservationLength int
	MaxReportLength      int
	logger               *log.Logger
	mu                   sync.Mutex
	collection           []OCRObservation
	completeReports      [][]OCRReport
}

func NewOCRController(round time.Duration, rnds int, logger *log.Logger, rcvrs ...*OCRReceiver) *OCRController {
	return &OCRController{
		RoundTime:       round,
		Queries:         make(chan OCRQuery, 1),
		Observations:    make(chan OCRObservation, len(rcvrs)),
		Reports:         make(chan OCRReport, len(rcvrs)),
		Stop:            make(chan struct{}, 1),
		Receivers:       rcvrs,
		MaxRounds:       rnds,
		logger:          logger,
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

	var cancel context.CancelFunc = func() {}
	if c.QueryTime > 0 {
		call.Context, cancel = context.WithTimeout(call.Context, c.QueryTime)
	}

	go func(r *OCRReceiver) {
		defer wg.Done()

		select {
		case r.Init <- OCRCall[struct{}]{OCRCallBase: call, Data: struct{}{}}:
			c.logger.Printf("init call sent to %s", r.Name)
			return
		case <-ctx.Done():
			cancel()
			return
		}
	}(c.Receivers[0])

	wg.Wait()
}

func (c *OCRController) sendQueries(ctx context.Context, call OCRCallBase) {
	var wg sync.WaitGroup
	var cancel context.CancelFunc = func() {}

	select {
	case q := <-c.Queries:
		c.logger.Printf("received query from leader")
		if len([]byte(q)) > c.MaxQueryLength {
			c.logger.Printf("[error] max query length exceeded")
		}

		for _, rec := range c.Receivers {
			wg.Add(1)

			if c.ObservationTime > 0 {
				call.Context, cancel = context.WithTimeout(call.Context, c.ObservationTime)
			}

			go func(r *OCRReceiver) {
				defer wg.Done()

				select {
				case r.Query <- OCRCall[OCRQuery]{OCRCallBase: call, Data: q}:
					c.logger.Printf("sent query to %s", r.Name)
					return
				case <-ctx.Done():
					cancel()
					return
				}
			}(rec)
		}

		wg.Wait()
	case <-ctx.Done():
		cancel()
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
				if len([]byte(o)) > c.MaxObservationLength {
					c.logger.Printf("[error] max observation length exceeded from %s", c.Receivers[idx].Name)
				}

				c.logger.Printf("received observation from %s", c.Receivers[idx].Name)
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
	var cancel context.CancelFunc = func() {}

	// send observations to all nodes
	c.mu.Lock()
	copy := c.collection
	c.mu.Unlock()

	for _, rec := range c.Receivers {
		wg.Add(1)

		if c.ReportTime > 0 {
			call.Context, cancel = context.WithTimeout(call.Context, c.ReportTime)
		}

		go func(r *OCRReceiver) {
			defer wg.Done()

			select {
			case r.Observations <- OCRCall[[]OCRObservation]{OCRCallBase: call, Data: copy}:
				c.logger.Printf("sent observations to %s", r.Name)
				return
			case <-ctx.Done():
				cancel()
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
				if len([]byte(r)) > c.MaxReportLength {
					c.logger.Printf("[error] max report length exceeded from %s", c.Receivers[i].Name)
				}
				c.logger.Printf("report received from %s", c.Receivers[i].Name)
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
	lastRound := len(c.completeReports) - 1
	hasReports := len(c.completeReports[lastRound])
	collapsed, err := c.collapseReports(c.completeReports[lastRound])
	c.mu.Unlock()

	if err != nil {
		panic(err)
	}

	if hasReports > 0 {
		rpt := collapsed

		for _, rec := range c.Receivers {
			wg.Add(1)
			go func(r *OCRReceiver) {
				defer wg.Done()

				select {
				case r.Report <- OCRCall[OCRReport]{OCRCallBase: call, Data: rpt}:
					c.logger.Printf("sent final report to %s", r.Name)
					return
				case <-ctx.Done():
					return
				}
			}(rec)
		}
	}

	wg.Wait()
}

func (c *OCRController) collapseReports(reports []OCRReport) (OCRReport, error) {
	// TODO: naive implementation assumes all reports are the same and sends the first
	if len(reports) == 0 {
		return nil, fmt.Errorf("cannot collapse reports of length 0")
	}

	return reports[0], nil
}

func (c *OCRController) stopReceivers() {
	for _, rec := range c.Receivers {
		go func(r *OCRReceiver) {
			r.Stop <- struct{}{}
		}(rec)
	}
}

func (c *OCRController) Start(ctx context.Context) chan struct{} {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println(err)
			debug.PrintStack()
		}
	}()
	done := make(chan struct{}, 1)

	go func() {
		//tkr := time.NewTicker(c.RoundTime)
		tkr := 0 * time.Second
		iterations := 0

		c.logger.Printf("starting OCR controller with round time of %d seconds", c.RoundTime/time.Second)

		for {
			t := time.NewTimer(tkr)

			select {
			case <-c.Stop:
				c.logger.Printf("receivers stopping")
				c.stopReceivers()
				c.logger.Printf("receivers stopped")
				t.Stop()
				done <- struct{}{}
				return
			case <-t.C:
				iterations++
				roundCtx, cancel := context.WithDeadline(ctx, time.Now().Add(c.RoundTime-(10*time.Millisecond)))
				base := OCRCallBase{Context: roundCtx, Round: iterations, Epoch: 1}

				c.logger.Printf("-----> round %d begins", iterations)

				// send call to Query to begin round
				c.sendInitCall(roundCtx, base)

				// send Query to all recievers to begin creating Observations
				c.sendQueries(roundCtx, base)

				// collect all observations from receivers
				c.collectObservations(roundCtx)

				// send observations to all receivers
				c.sendObservations(roundCtx, base)

				c.collectReports(roundCtx, base)

				c.logger.Printf("<----- round %d ends", iterations)
				if c.MaxRounds > 0 && iterations == c.MaxRounds {
					c.logger.Printf("max rounds encountered; terminating process")
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

func (c *OCRController) WriteReports(path string) {
	c.mu.Lock()
	for i, nodeReports := range c.completeReports {
		for j, r := range nodeReports {
			name := fmt.Sprintf("node_%d_round_%d", i+1, j+1)

			dst := make([]byte, hex.EncodedLen(len(r)))
			hex.Encode(dst, r)

			f, err := os.OpenFile(fmt.Sprintf("%s/%s", path, name), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
			if err != nil {
				c.logger.Println(err.Error())
				f.Close()
				continue
			}

			_, err = f.Write(dst)
			if err != nil {
				c.logger.Println(err.Error())
			}

			f.Close()
		}
	}
	c.mu.Unlock()
}
