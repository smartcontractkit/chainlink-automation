package upkeep

import (
	"log"
	"math/big"
	"runtime"
	"sync"

	"github.com/smartcontractkit/ocr2keepers/cmd/simulator/simulate/chain"
)

// LogTriggerTracker ...
type LogTriggerTracker struct {
	// provided dependencies
	listener *chain.Listener
	active   *ActiveTracker
	performs *PerformTracker
	logger   *log.Logger

	// internal state props
	mu        sync.RWMutex
	triggered []triggeredUpkeep
	read      []triggeredUpkeep

	// service values
	chDone chan struct{}
}

// NewLogTriggerTracker ...
func NewLogTriggerTracker(
	listener *chain.Listener,
	active *ActiveTracker,
	performs *PerformTracker,
	logger *log.Logger,
) *LogTriggerTracker {
	src := &LogTriggerTracker{
		listener:  listener,
		active:    active,
		performs:  performs,
		logger:    log.New(logger.Writer(), "[log-trigger-tracker] ", log.Ldate|log.Ltime|log.Lshortfile),
		triggered: make([]triggeredUpkeep, 0),
		read:      make([]triggeredUpkeep, 0),
		chDone:    make(chan struct{}),
	}

	go src.run()

	runtime.SetFinalizer(src, func(srv *LogTriggerTracker) { srv.stop() })

	return src
}

// GetOnce will return a set of triggered upkeeps exactly once. Subsequent calls
// are guaranteed to not receive the same upkeeps again. GetOnce will not check
// against performs to validate a triggered upkeep is performable before
// returning it in results.
func (ltt *LogTriggerTracker) GetOnce() []triggeredUpkeep {
	ltt.mu.Lock()
	defer ltt.mu.Unlock()

	once := make([]triggeredUpkeep, len(ltt.triggered))

	copy(once, ltt.triggered)

	ltt.read = append(ltt.read, ltt.triggered...)
	ltt.triggered = make([]triggeredUpkeep, 0)

	return once
}

// GetAfter will return triggered upkeeps older than the provided block number
// and that have not been viewed by GetOnce. Subsequent calls may return the
// same results. GetAfter will validate that a triggered upkeep has not yet been
// performed.
func (ltt *LogTriggerTracker) GetAfter(number *big.Int) []triggeredUpkeep {
	ltt.mu.RLock()
	defer ltt.mu.RUnlock()

	if len(ltt.read) == 0 {
		return nil
	}

	output := make([]triggeredUpkeep, 0, len(ltt.read))

	// triggered upkeeps are stored oldest to newest. start at the end and
	// read backward
	for x := len(ltt.read) - 1; x >= 0; x-- {
		// only the payload workID is needed from this so the latest block number
		// is fine to provide to the payload build function
		payload := makeLogPayloadFromUpkeep(ltt.read[x], ltt.active.GetLatestBlock())
		ok := ltt.performs.IsWorkIDPerformed(payload.WorkID)

		if ltt.read[x].blockNumber.Cmp(number) < 0 && !ok {
			output = append(output, ltt.read[x])
		}
	}

	return output
}

func (ltt *LogTriggerTracker) run() {
	chEvents := ltt.listener.Subscribe(chain.LogTriggerChannel)

	for {
		select {
		case event := <-chEvents:
			switch evt := event.Event.(type) {
			case chain.Log:
				ltt.createLogUpkeeps(event.BlockNumber, event.BlockHash, evt)
			}
		case <-ltt.chDone:
			return
		}
	}
}

func (ltt *LogTriggerTracker) stop() {
	close(ltt.chDone)
}

func (ltt *LogTriggerTracker) createLogUpkeeps(number *big.Int, hash [32]byte, chainLog chain.Log) {
	upkeeps := ltt.active.GetAllByType(chain.LogTriggerType)
	if len(upkeeps) == 0 {
		return
	}

	ltt.mu.Lock()
	defer ltt.mu.Unlock()

	for _, upkeep := range upkeeps {
		if logTriggersUpkeep(chainLog, upkeep) {
			ltt.triggered = append(ltt.triggered, triggeredUpkeep{
				upkeep:      upkeep,
				blockNumber: number,
				blockHash:   hash,
				chainLog:    chainLog,
			})
		}
	}
}

func logTriggersUpkeep(chainLog chain.Log, upkeep chain.SimulatedUpkeep) bool {
	return chainLog.TriggerValue == upkeep.TriggeredBy
}

type triggeredUpkeep struct {
	upkeep      chain.SimulatedUpkeep
	blockNumber *big.Int
	blockHash   [32]byte
	chainLog    chain.Log
}
