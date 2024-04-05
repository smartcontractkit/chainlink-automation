package hooks

import (
	"bytes"
	"fmt"
	"log"
	"sort"
	"sync"

	"github.com/smartcontractkit/chainlink-common/pkg/types/automation"

	ocr2keepersv3 "github.com/smartcontractkit/chainlink-automation/pkg/v3"
	"github.com/smartcontractkit/chainlink-automation/pkg/v3/random"
	"github.com/smartcontractkit/chainlink-automation/pkg/v3/telemetry"
	"github.com/smartcontractkit/chainlink-automation/pkg/v3/types"
)

type stagingHookState struct {
	lastRandSrc [16]byte
	shuffledIDs map[string]string
}

type AddFromStagingHook struct {
	store  types.ResultStore
	logger *log.Logger
	coord  types.Coordinator
	state  stagingHookState
	lock   sync.Mutex
}

func NewAddFromStagingHook(store types.ResultStore, coord types.Coordinator, logger *log.Logger) AddFromStagingHook {
	return AddFromStagingHook{
		store:  store,
		coord:  coord,
		logger: log.New(logger.Writer(), fmt.Sprintf("[%s | build hook:add-from-staging]", telemetry.ServiceName), telemetry.LogPkgStdFlags),
		state: stagingHookState{
			shuffledIDs: make(map[string]string),
		},
	}
}

// RunHook adds results from the store to the observation.
// It sorts by a shuffled workID. workID for all items is shuffled using a pseudorandom source
// that is the same across all nodes for a given round. This ensures that all nodes try to
// send the same subset of workIDs if they are available, while giving different priority
// to workIDs in different rounds.
func (hook *AddFromStagingHook) RunHook(obs *ocr2keepersv3.AutomationObservation, limit int, rSrc [16]byte) error {
	results, err := hook.store.View()
	if err != nil {
		return err
	}
	results, err = hook.coord.FilterResults(results)
	if err != nil {
		return err
	}

	results = hook.orderResults(results, rSrc)
	n := len(results)
	if n > limit {
		results = results[:limit]
	}
	hook.logger.Printf("adding %d results to observation, out of %d available results", len(results), n)
	obs.Performable = append(obs.Performable, results...)

	return nil
}

func (hook *AddFromStagingHook) orderResults(results []automation.CheckResult, rSrc [16]byte) []automation.CheckResult {
	hook.lock.Lock()
	defer hook.lock.Unlock()

	shuffledIDs := hook.updateShuffledIDs(results, rSrc)
	// sort by the shuffled workID
	sort.Slice(results, func(i, j int) bool {
		return shuffledIDs[results[i].WorkID] < shuffledIDs[results[j].WorkID]
	})

	return results
}

func (hook *AddFromStagingHook) updateShuffledIDs(results []automation.CheckResult, rSrc [16]byte) map[string]string {
	// once the random source changes, the workIDs needs to be shuffled again with the new source
	if !bytes.Equal(hook.state.lastRandSrc[:], rSrc[:]) {
		hook.state.lastRandSrc = rSrc
		hook.state.shuffledIDs = make(map[string]string)
	}

	for _, result := range results {
		if _, ok := hook.state.shuffledIDs[result.WorkID]; !ok {
			hook.state.shuffledIDs[result.WorkID] = random.ShuffleString(result.WorkID, rSrc)
		}
	}

	return hook.state.shuffledIDs
}
