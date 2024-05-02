package hooks

import (
	"bytes"
	"encoding/json"
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
func (hook *AddFromStagingHook) RunHook(obs *ocr2keepersv3.AutomationObservation, rSrc [16]byte) error {
	results, err := hook.store.View()
	if err != nil {
		return err
	}
	results, err = hook.coord.FilterResults(results)
	if err != nil {
		return err
	}

	results = hook.sorter.orderResults(results, rSrc)

	count := hook.addByEstimatesAggressive(obs, results)

	hook.logger.Printf("adding %d results to observation", count)

	if count < len(results) {
		hook.logger.Printf("skipped %d available results in staging", len(results)-count)
	}

	return nil
}

func (hook *AddFromStagingHook) addByEstimates(obs *ocr2keepersv3.AutomationObservation, results []automation.CheckResult) int {
	added := 0
	for _, result := range results {
		obs.Performable = append(obs.Performable, result)
		added++

		if obs.Length() > ocr2keepersv3.MaxObservationLength {
			obs.Performable = obs.Performable[:len(obs.Performable)-1]
			added--
			break
		}
	}

	return added
}

func (hook *AddFromStagingHook) addByEstimatesAggressive(obs *ocr2keepersv3.AutomationObservation, results []automation.CheckResult) int {
	added := 0
	i := 0
	for ; i < len(results); i++ {
		result := results[i]
		obs.Performable = append(obs.Performable, result)
		added++

		if obs.Length() > ocr2keepersv3.MaxObservationLength {
			obs.Performable = obs.Performable[:len(obs.Performable)-1]
			added--
			break
		}
	}

	// At this point, we've only compared raw json with the max observation length.
	// We want to try and squeeze as much space as we can out of the observation length limit,
	// so we try to estimate how many more performables we can add, and after adding all the
	// additional performables, we marshal the observation and compare the observation
	// length in bytes with the max observation length. If the serialized observation is above the limit,
	// we repeatedely remove performables one at a time, and marshal the observation, until the length
	// of the observation is under the limit.
	if added > 0 {
		b, _ := json.Marshal(obs)

		// determine how much space we have left within the max observation length
		bytesRemaining := ocr2keepersv3.MaxObservationLength - len(b)

		// determine roughly how many bytes each performable has taken up so far
		bytesPerPerformable := len(b) / added

		// based on the average performable byte size, and the remaining space within the observation
		// length limit, calculate the number of records we think we can add
		recordsToAdd := bytesRemaining / bytesPerPerformable

		// add more performables to the observation, continuing where the original add operation left off
		for recordsAdded := 0; recordsAdded < recordsToAdd && i < len(results); recordsAdded++ {
			obs.Performable = append(obs.Performable, results[i])
			added++
			i++
		}

		// now, marshal the observation again, check if we've exceeded the observation length, and if we have,
		// remove one performable from the observation, and repeat until the serialized observation is under the
		// max observation length
		for b, _ := json.Marshal(obs); len(b) > ocr2keepersv3.MaxObservationLength; b, _ = json.Marshal(obs) {
			obs.Performable = obs.Performable[:len(obs.Performable)-1]
			added--
		}
	}

	return added
}

func (hook *AddFromStagingHook) addByJSON(obs *ocr2keepersv3.AutomationObservation, results []automation.CheckResult) int {
	added := 0

	for _, result := range results {
		obs.Performable = append(obs.Performable, result)
		added++

		if b, _ := obs.Encode(); len(b) > ocr2keepersv3.MaxObservationLength {
			obs.Performable = obs.Performable[:len(obs.Performable)-1]
			added--
			break
		}
	}

	return added
}

type stagedResultSorter struct {
	lastRandSrc [16]byte
	shuffledIDs map[string]string
	lock        sync.Mutex
}

// orderResults orders the results by the shuffled workID
func (sorter *stagedResultSorter) orderResults(results []automation.CheckResult, rSrc [16]byte) []automation.CheckResult {
	sorter.lock.Lock()
	defer sorter.lock.Unlock()

	shuffledIDs := sorter.updateShuffledIDs(results, rSrc)
	// sort by the shuffled workID
	sort.Slice(results, func(i, j int) bool {
		return shuffledIDs[results[i].WorkID] < shuffledIDs[results[j].WorkID]
	})

	return results
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
