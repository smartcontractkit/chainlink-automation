package upkeep

import (
	"encoding/hex"
	"log"
	"math/big"
	"runtime"
	"sync"

	"github.com/smartcontractkit/ocr2keepers/cmd/simv3/simulator/chain"
)

type ActiveTracker struct {
	// provided dependencies
	listener *chain.Listener
	logger   *log.Logger

	// internal state props
	mu       sync.RWMutex
	active   map[chain.UpkeepType][]string    // maps types to a list of UpkeepID as hex values
	idLookup map[string]chain.SimulatedUpkeep // maps UpkeepID as a hex value to a simulated upkeep
	latest   chain.Block

	// service values
	chDone chan struct{}
}

func NewActiveTracker(listener *chain.Listener, logger *log.Logger) *ActiveTracker {
	src := &ActiveTracker{
		listener: listener,
		logger:   log.New(logger.Writer(), "[active-upkeep-tracker] ", log.Ldate|log.Ltime|log.Lshortfile),
		active:   make(map[chain.UpkeepType][]string),
		idLookup: make(map[string]chain.SimulatedUpkeep),
		latest: chain.Block{
			Number: big.NewInt(0),
		},
		chDone: make(chan struct{}),
	}

	go src.run()

	runtime.SetFinalizer(src, func(srv *ActiveTracker) { srv.stop() })

	return src
}

func (at *ActiveTracker) GetAllByType(upkeepType chain.UpkeepType) []chain.SimulatedUpkeep {
	at.mu.RLock()
	defer at.mu.RUnlock()

	ids, ok := at.active[upkeepType]
	if !ok {
		return nil
	}

	upkeeps := make([]chain.SimulatedUpkeep, 0, len(ids))

	for _, id := range ids {
		if upkeep, ok := at.idLookup[id]; ok {
			upkeeps = append(upkeeps, upkeep)
		}
	}

	return upkeeps
}

// GetByID returns a simulated upkeep identified by a unique hash value
func (at *ActiveTracker) GetByUpkeepID(uniqueID [32]byte) (chain.SimulatedUpkeep, bool) {
	at.mu.RLock()
	defer at.mu.RUnlock()

	key := hex.EncodeToString(uniqueID[:])

	if upkeep, ok := at.idLookup[key]; ok {
		return upkeep, true
	}

	return chain.SimulatedUpkeep{}, false
}

func (at *ActiveTracker) GetLatestBlock() chain.Block {
	at.mu.RLock()
	defer at.mu.RUnlock()

	return at.latest
}

func (at *ActiveTracker) run() {
	chEvents := at.listener.Subscribe(chain.CreateUpkeepChannel, chain.BlockChannel)

	for {
		select {
		case event := <-chEvents:
			switch evt := event.Event.(type) {
			case chain.UpkeepCreatedTransaction:
				at.addSimulatedUpkeep(evt.Upkeep)
			case chain.Block:
				at.setLatestBlock(evt)
			}
		case <-at.chDone:
			return
		}
	}
}

func (at *ActiveTracker) stop() {
	close(at.chDone)
}

func (at *ActiveTracker) addSimulatedUpkeep(upkeep chain.SimulatedUpkeep) {
	at.mu.Lock()
	defer at.mu.Unlock()

	upkeepIDs, ok := at.active[upkeep.Type]
	if !ok {
		upkeepIDs = []string{}
	}

	key := hex.EncodeToString(upkeep.UpkeepID[:])

	upkeepIDs = append(upkeepIDs, key)

	at.active[upkeep.Type] = upkeepIDs
	at.idLookup[key] = upkeep
}

func (at *ActiveTracker) setLatestBlock(block chain.Block) {
	at.mu.Lock()
	defer at.mu.Unlock()

	at.latest = block
}
