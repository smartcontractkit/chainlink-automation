package upkeep

import (
	"fmt"
	"log"
	"math/big"
	"runtime"
	"sync"

	"github.com/smartcontractkit/ocr2keepers/cmd/simv3/simulator/chain"
	"github.com/smartcontractkit/ocr2keepers/cmd/simv3/util"
	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg/v3/types"
)

type PerformTracker struct {
	// provided dependencies
	listener *chain.Listener
	logger   *log.Logger

	// internal state props
	mu           sync.RWMutex
	conditionals map[string][]*big.Int // maps UpkeepID to performed in block for conditionals
	performed    map[string]bool       // maps WorkIDs to performed state

	// service values
	chDone chan struct{}
}

// NewPerformTracker ...
func NewPerformTracker(listener *chain.Listener, logger *log.Logger) *PerformTracker {
	src := &PerformTracker{
		listener:     listener,
		logger:       log.New(logger.Writer(), "[perform-tracker] ", log.Ldate|log.Ltime|log.Lshortfile),
		conditionals: make(map[string][]*big.Int),
		performed:    make(map[string]bool),
		chDone:       make(chan struct{}),
	}

	go src.run()

	runtime.SetFinalizer(src, func(srv *PerformTracker) { srv.stop() })

	return src
}

func (pt *PerformTracker) PerformsForUpkeepID(upkeepID string) []*big.Int {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	if performs, ok := pt.conditionals[upkeepID]; ok {
		return performs
	}

	return nil
}

func (pt *PerformTracker) IsWorkIDPerformed(workID string) bool {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	if ok, exists := pt.performed[workID]; exists && ok {
		return true
	}
	fmt.Println(workID)

	return false
}

func (pt *PerformTracker) run() {
	chEvents := pt.listener.Subscribe(chain.PerformUpkeepChannel)

	for {
		select {
		case event := <-chEvents:
			switch evt := event.Event.(type) {
			case chain.PerformUpkeepTransaction:
				pt.registerTransmitted(evt.Transmits...)
			}
		case <-pt.chDone:
			return
		}
	}
}

func (pt *PerformTracker) stop() {
	close(pt.chDone)
}

func (pt *PerformTracker) registerTransmitted(transmits ...chain.TransmitEvent) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	for _, transmit := range transmits {
		trResults, err := util.DecodeCheckResultsFromReportBytes(transmit.Report)
		if err != nil {
			pt.logger.Println(err)

			continue
		}

		for _, result := range trResults {
			key := result.UpkeepID.String()

			pt.performed[result.WorkID] = true

			if util.GetUpkeepType(result.UpkeepID) == ocr2keepers.ConditionTrigger {
				performs, ok := pt.conditionals[key]
				if !ok {
					performs = []*big.Int{}
				}

				pt.conditionals[key] = append(performs, transmit.BlockNumber)
			}
		}
	}
}
