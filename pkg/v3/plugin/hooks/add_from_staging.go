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

type AddFromStagingHook struct {
	store  types.ResultStore
	logger *log.Logger
	coord  types.Coordinator
	sorter stagedResultSorter
}

func NewAddFromStagingHook(store types.ResultStore, coord types.Coordinator, logger *log.Logger) AddFromStagingHook {
	return AddFromStagingHook{
		store:  store,
		coord:  coord,
		logger: log.New(logger.Writer(), fmt.Sprintf("[%s | build hook:add-from-staging]", telemetry.ServiceName), telemetry.LogPkgStdFlags),
		sorter: stagedResultSorter{
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

	n := len(results)
	results = hook.sorter.orderResults(results, limit, rSrc)
	if n > limit {
		hook.logger.Printf("skipped %d staged results", n-limit)
	}
	hook.logger.Printf("adding %d results to observation", len(results))
	obs.Performable = append(obs.Performable, results...)

	return nil
}

type stagedResultSorter struct {
	lastRandSrc [16]byte
	shuffledIDs map[string]string
	lock        sync.Mutex
}

// orderResults orders the results by the shuffled workID and returns the first (limit) elements.
func (sorter *stagedResultSorter) orderResults(results []automation.CheckResult, limit int, rSrc [16]byte) []automation.CheckResult {
	sorter.lock.Lock()
	defer sorter.lock.Unlock()

	shuffledIDs := sorter.updateShuffledIDs(results, rSrc)
	// sort by the shuffled workID
	// if the limit is greater than the number of results, sort the whole slice
	if limit >= len(results) {
		sort.Slice(results, func(i, j int) bool {
			return shuffledIDs[results[i].WorkID] < shuffledIDs[results[j].WorkID]
		})
		return results
	}
	// otherwise, sort only the first limit elements to be more efficient than sorting the whole slice
	return sorter.partialSort(results, shuffledIDs, limit)
}

// partialSort sorts the first limit elements of the results slice.
// using bubble sort as it allows for early termination when the slice is sorted up to the limit.
// NOTE: this function assumes len(results) > limit and that the shuffledIDs are already populated.
func (sorter *stagedResultSorter) partialSort(results []automation.CheckResult, shuffledIDs map[string]string, limit int) []automation.CheckResult {
	for i := 0; i < limit; i++ {
		for j := i + 1; j < len(results); j++ {
			if shuffledIDs[results[i].WorkID] > shuffledIDs[results[j].WorkID] {
				results[i], results[j] = results[j], results[i]
			}
		}
	}
	return results[:limit]
}

// updateShuffledIDs updates the shuffledIDs cache with the new random source or items.
// NOTE: This function is not thread-safe and should be called with a lock
func (sorter *stagedResultSorter) updateShuffledIDs(results []automation.CheckResult, rSrc [16]byte) map[string]string {
	// once the random source changes, the workIDs needs to be shuffled again with the new source
	if !bytes.Equal(sorter.lastRandSrc[:], rSrc[:]) {
		sorter.lastRandSrc = rSrc
		sorter.shuffledIDs = make(map[string]string)
	}

	for _, result := range results {
		if _, ok := sorter.shuffledIDs[result.WorkID]; !ok {
			sorter.shuffledIDs[result.WorkID] = random.ShuffleString(result.WorkID, rSrc)
		}
	}

	return sorter.shuffledIDs
}
