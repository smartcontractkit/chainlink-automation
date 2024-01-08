package hooks

import (
	"sort"

	ocr2keepersv3 "github.com/smartcontractkit/chainlink-automation/pkg/v3"
	"github.com/smartcontractkit/chainlink-automation/pkg/v3/random"
	"github.com/smartcontractkit/chainlink-automation/pkg/v3/telemetry"
	"github.com/smartcontractkit/chainlink-automation/pkg/v3/types"
	ocr2keepers "github.com/smartcontractkit/chainlink-automation/pkg/v3/types"
)

func NewAddFromStagingHook(store ocr2keepers.ResultStore, coord types.Coordinator, logger *telemetry.Logger) AddFromStagingHook {
	return AddFromStagingHook{
		store:  store,
		coord:  coord,
		logger: telemetry.WrapTelemetryLogger(logger, "build hook:add-from-staging"),
	}
}

type AddFromStagingHook struct {
	store  ocr2keepers.ResultStore
	logger *telemetry.Logger
	coord  types.Coordinator
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
	// creating a map to hold the shuffled workIDs
	shuffledIDs := make(map[string]string, len(results))
	for _, result := range results {
		shuffledIDs[result.WorkID] = random.ShuffleString(result.WorkID, rSrc)
	}
	// sort by the shuffled workID
	sort.Slice(results, func(i, j int) bool {
		return shuffledIDs[results[i].WorkID] < shuffledIDs[results[j].WorkID]
	})
	if len(results) > limit {
		results = results[:limit]
	}

	for _, result := range results {
		if err := hook.logger.Collect(result.WorkID, uint64(result.Trigger.BlockNumber), telemetry.ResultProposed); err != nil {
			hook.logger.Println(err.Error())
		}
	}

	hook.logger.Printf("adding %d results to observation", len(results))
	obs.Performable = append(obs.Performable, results...)

	return nil
}
