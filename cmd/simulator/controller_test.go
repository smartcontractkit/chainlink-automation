package main

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMockCollectObservations(t *testing.T) {
	r1 := NewOCRReceiver("1")
	r2 := NewOCRReceiver("2")

	duration := 100 * time.Millisecond
	c := NewOCRController(duration, 2, r1, r2)

	c.Observations <- OCRObservation([]byte("one"))
	c.Observations <- OCRObservation([]byte("two"))

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(10*time.Millisecond))
	c.collectObservations(ctx)
	cancel()

	assert.Contains(t, c.collection, OCRObservation([]byte("one")))
	assert.Contains(t, c.collection, OCRObservation([]byte("two")))
}

func TestMockSendObservations(t *testing.T) {
	r1 := NewOCRReceiver("1")
	r2 := NewOCRReceiver("2")

	duration := 100 * time.Millisecond
	c := NewOCRController(duration, 2, r1, r2)

	c.collection = []OCRObservation{OCRObservation([]byte("one")), OCRObservation([]byte("two"))}

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(200*time.Millisecond))
	base := OCRCallBase{Context: ctx, Round: 1, Epoch: 1}
	c.sendObservations(ctx, base)

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()

		select {
		case a := <-r1.Observations:
			assert.Equal(t, a.Data, c.collection)
		case <-ctx.Done():
			assert.Fail(t, "missing observation for r1")
			return
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		select {
		case a := <-r2.Observations:
			assert.Equal(t, a.Data, c.collection)
		case <-ctx.Done():
			assert.Fail(t, "missing observation for r2")
			return
		}
	}()

	wg.Wait()
	cancel()
}

func TestMockCollectReports(t *testing.T) {
	r1 := NewOCRReceiver("1")
	r2 := NewOCRReceiver("2")

	duration := 100 * time.Millisecond
	c := NewOCRController(duration, 2, r1, r2)

	c.Reports <- OCRReport([]byte("one"))
	c.Reports <- OCRReport([]byte("two"))

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(200*time.Millisecond))
	base := OCRCallBase{Context: ctx, Round: 1, Epoch: 1}
	c.collectReports(ctx, base)

	var wg sync.WaitGroup

	if len(c.completeReports) > 0 {
		assert.Contains(t, c.completeReports[0], OCRReport([]byte("one")))
		assert.Contains(t, c.completeReports[0], OCRReport([]byte("two")))
	} else {
		assert.Fail(t, "missing complete reports")
	}

	wg.Add(1)
	go func() {
		defer wg.Done()

		select {
		case a := <-r1.Report:
			assert.Equal(t, a.Data, c.completeReports[0][0])
		case <-ctx.Done():
			assert.Fail(t, "missing report for r1")
			return
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		select {
		case a := <-r2.Report:
			assert.Equal(t, a.Data, c.completeReports[0][0])
		case <-ctx.Done():
			assert.Fail(t, "missing report for r2")
			return
		}
	}()

	wg.Wait()
	cancel()
}
