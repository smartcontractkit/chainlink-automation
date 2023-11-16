package hooks

import (
	"fmt"
	"log"
	"sort"

	ocr2keepersv3 "github.com/smartcontractkit/chainlink-automation/pkg/v3"
	"github.com/smartcontractkit/chainlink-automation/pkg/v3/random"
	"github.com/smartcontractkit/chainlink-automation/pkg/v3/telemetry"
	"github.com/smartcontractkit/chainlink-automation/pkg/v3/types"
	ocr2keepers "github.com/smartcontractkit/chainlink-automation/pkg/v3/types"
)

func NewAddFromStagingHook(store ocr2keepers.ResultStore, coord types.Coordinator, logger *log.Logger) AddFromStagingHook {
	return AddFromStagingHook{
		store:  store,
		coord:  coord,
		logger: log.New(logger.Writer(), fmt.Sprintf("[%s | build hook:add-from-staging]", telemetry.ServiceName), telemetry.LogPkgStdFlags),
	}
}

type AddFromStagingHook struct {
	store  ocr2keepers.ResultStore
	logger *log.Logger
	coord  types.Coordinator
}

// RunHook adds results from the store to the observation.
// It sorts by a shuffled workID. workID for all items is shuffled using a pseduorandom souce
// that is the same across all nodes for a given round. This ensures that all nodes try to
// send the same subset of workIDs if they are available, while giving different priority
// to workIDs in different rounds.
func (hook *AddFromStagingHook) RunHook(obs *ocr2keepersv3.AutomationObservation, limit int, rSrc [16]byte) error {
	storeResults, err := hook.store.View()
	if err != nil {
		return err
	}
	storeResults, err = hook.coord.FilterResults(storeResults)
	if err != nil {
		return err
	}
	// create a slice of shuffled workID strings, i.e. calling random.ShuffleString once per workID
	shuffledStrings := make([]shuffledString, len(storeResults))
	for i, result := range storeResults {
		shuffledStrings[i] = shuffledString{
			val:       random.ShuffleString(result.WorkID, rSrc),
			origIndex: i,
		}
	}
	// sort by the shuffled workID
	sort.Slice(shuffledStrings, func(i, j int) bool {
		return shuffledStrings[i].val < shuffledStrings[j].val
	})
	if len(storeResults) > limit {
		shuffledStrings = shuffledStrings[:limit]
	}
	// create a slice of results in the order of the shuffled workIDs
	results := make([]ocr2keepers.CheckResult, len(shuffledStrings))
	for i, shuffled := range shuffledStrings {
		results[i] = storeResults[shuffled.origIndex]
	}

	hook.logger.Printf("adding %d results to observation", len(results))
	obs.Performable = append(obs.Performable, results...)

	return nil
}

// shuffledString represents a string that has been shuffled using a pseduorandom source
type shuffledString struct {
	val       string
	origIndex int
}
