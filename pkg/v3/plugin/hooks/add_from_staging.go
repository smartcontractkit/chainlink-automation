package hooks

import (
	"fmt"
	"log"
	"sort"
	"sync"

	ocr2keepersv3 "github.com/smartcontractkit/chainlink-automation/pkg/v3"
	"github.com/smartcontractkit/chainlink-automation/pkg/v3/random"
	"github.com/smartcontractkit/chainlink-automation/pkg/v3/telemetry"
	"github.com/smartcontractkit/chainlink-automation/pkg/v3/types"
)

var (
	maxCacheSize = 20_000
)

func NewAddFromStagingHook(store types.ResultStore, coord types.Coordinator, logger *log.Logger) AddFromStagingHook {
	return AddFromStagingHook{
		store:            store,
		coord:            coord,
		logger:           log.New(logger.Writer(), fmt.Sprintf("[%s | build hook:add-from-staging]", telemetry.ServiceName), telemetry.LogPkgStdFlags),
		shuffledIDsCache: make(map[string]string),
	}
}

type AddFromStagingHook struct {
	store  types.ResultStore
	logger *log.Logger
	coord  types.Coordinator

	shuffledIDsCache map[string]string
	suffledIDsLock   sync.Mutex
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
	n := len(results)
	hook.suffledIDsLock.Lock()
	for _, result := range results {
		if _, ok := hook.shuffledIDsCache[result.WorkID]; !ok {
			hook.shuffledIDsCache[result.WorkID] = random.ShuffleString(result.WorkID, rSrc)
		}
	}
	// sort by the shuffled workID
	sort.Slice(results, func(i, j int) bool {
		return hook.shuffledIDsCache[results[i].WorkID] < hook.shuffledIDsCache[results[j].WorkID]
	})
	if len(hook.shuffledIDsCache) > maxCacheSize {
		// TODO: find a better way to clean the cache
		hook.shuffledIDsCache = make(map[string]string)
	}
	if len(results) > limit {
		results = results[:limit]
	}
	hook.logger.Printf("adding %d results to observation, out of %d available results", len(results), n)
	obs.Performable = append(obs.Performable, results...)

	return nil
}
