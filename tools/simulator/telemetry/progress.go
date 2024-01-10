package telemetry

import (
	"io"
	"sync/atomic"
	"time"

	"github.com/jedib0t/go-pretty/v6/progress"
)

const (
	trackerProgressCheck = 100 * time.Millisecond
)

type ProgressTelemetry struct {
	success      atomic.Bool
	failed       atomic.Int32
	writer       progress.Writer
	incrementMap map[string]chan int64
	chDone       chan struct{}
	chComplete   chan struct{}
}

func NewProgressTelemetry(wOutput io.Writer) *ProgressTelemetry {
	writer := progress.NewWriter()

	writer.SetOutputWriter(wOutput)

	writer.LengthDone()
	writer.SetAutoStop(false)
	writer.SetTrackerLength(25)
	writer.SetMessageWidth(44)
	writer.SetSortBy(progress.SortByPercentDsc)
	writer.SetStyle(progress.StyleDefault)
	writer.SetTrackerPosition(progress.PositionRight)
	writer.SetUpdateFrequency(time.Millisecond * 100)

	writer.Style().Colors = progress.StyleColorsExample
	writer.Style().Options.PercentFormat = "%4.1f%%"
	writer.Style().Visibility.ETA = true
	writer.Style().Visibility.ETAOverall = true
	writer.Style().Visibility.Percentage = true
	writer.Style().Visibility.Speed = true
	writer.Style().Visibility.SpeedOverall = true
	writer.Style().Visibility.Time = true
	writer.Style().Visibility.TrackerOverall = true
	writer.Style().Visibility.Value = true
	writer.Style().Visibility.Pinned = true

	return &ProgressTelemetry{
		writer:       writer,
		incrementMap: make(map[string]chan int64),
		chDone:       make(chan struct{}),
		chComplete:   make(chan struct{}),
	}
}

func (t *ProgressTelemetry) Register(namespace string, total int64) error {
	chIncrements := make(chan int64, 100)
	t.incrementMap[namespace] = chIncrements

	go t.track(namespace, total, chIncrements)

	return nil
}

func (t *ProgressTelemetry) Increment(namespace string, count int64) {
	if chIncrements, exists := t.incrementMap[namespace]; exists {
		go func() {
			chIncrements <- count
		}()
	}
}

func (t *ProgressTelemetry) AllProgressComplete() bool {
	<-t.chComplete

	return t.success.Load()
}

func (t *ProgressTelemetry) Start() {
	go t.writer.Render()
	go t.checkProgress()
}

func (t *ProgressTelemetry) Close() error {
	close(t.chDone)

	return nil
}

func (t *ProgressTelemetry) track(namespace string, total int64, chIncrements chan int64) {
	var negativeAssert bool

	// for total == 0, the tracker is a negative assertion and 0 increments is a positive case
	// to make the tracker function well, set the expected increment to 1 and only increment on completion
	if total == 0 {
		negativeAssert = true
	}

	tracker := progress.Tracker{
		Message:    namespace,
		Total:      total,
		Units:      progress.UnitsDefault,
		DeferStart: true,
	}

	t.writer.AppendTracker(&tracker)

	for !tracker.IsDone() {
		select {
		case increment := <-chIncrements:
			if negativeAssert {
				tracker.MarkAsErrored()
				t.failed.Add(1)
				break
			}

			tracker.Increment(increment)
		case <-t.chDone:
			if negativeAssert {
				tracker.MarkAsDone()
			}

			if tracker.Value() != total {
				t.failed.Add(1)
				tracker.MarkAsErrored()
			}
		}
	}
}

func (t *ProgressTelemetry) checkProgress() {
	ticker := time.NewTicker(trackerProgressCheck)

	for t.writer.IsRenderInProgress() {
		select {
		case <-ticker.C:
			if t.writer.LengthActive() == 0 {
				t.writer.Stop()
			}
		case <-t.chDone:
			// wait for all trackers to complete
			time.Sleep(500 * time.Millisecond)

			ticker.Stop()
			t.writer.Stop()
		}
	}

	t.success.Store(t.writer.Length() == t.writer.LengthDone() && t.failed.Load() == 0)

	close(t.chComplete)
}
