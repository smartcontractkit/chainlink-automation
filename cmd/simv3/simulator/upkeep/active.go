package upkeep

import (
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
	active   map[chain.UpkeepType][]*big.Int
	idLookup map[*big.Int]chain.SimulatedUpkeep

	// service values
	chDone chan struct{}
}

func NewActiveTracker(listener *chain.Listener, logger *log.Logger) *ActiveTracker {
	src := &ActiveTracker{
		listener: listener,
		logger:   log.New(logger.Writer(), "[active-upkeep-tracker]", log.LstdFlags),
		active:   make(map[chain.UpkeepType][]*big.Int),
		idLookup: make(map[*big.Int]chain.SimulatedUpkeep),
		chDone:   make(chan struct{}),
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

func (at *ActiveTracker) GetByID(numericID *big.Int) (chain.SimulatedUpkeep, bool) {
	at.mu.RLock()
	defer at.mu.RUnlock()

	if upkeep, ok := at.idLookup[numericID]; ok {
		return upkeep, true
	}

	return chain.SimulatedUpkeep{}, false
}

func (at *ActiveTracker) run() {
	chEvents := at.listener.Subscribe(chain.CreateUpkeepChannel)

	for {
		select {
		case event := <-chEvents:
			switch evt := event.Event.(type) {
			case chain.UpkeepCreatedTransaction:
				at.addSimulatedUpkeep(evt.Upkeep)
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
		upkeepIDs = []*big.Int{}
	}

	upkeepIDs = append(upkeepIDs, upkeep.ID)

	at.active[upkeep.Type] = upkeepIDs
	at.idLookup[upkeep.ID] = upkeep
}
