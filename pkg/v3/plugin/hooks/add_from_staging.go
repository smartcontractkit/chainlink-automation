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
// It sorts by a shuffled workID. workID for all items is shuffled using a pseudorandom source
// that is the same across all nodes for a given round. This ensures that all nodes try to
// send the same subset of workIDs if they are available, while giving different priority
// to workIDs in different rounds.
func (hook *AddFromStagingHook) RunHook(obs *ocr2keepersv3.AutomationObservation, sizeLimit, limit int, rSrc [16]byte) error {
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

	observationSize := obs.BlockHistory.Size()
	for _, proposal := range obs.UpkeepProposals {
		observationSize += proposal.Size()
	}
	for _, result := range obs.Performable {
		observationSize += result.Size()
	}

	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}
	added := 0
	for _, result := range results {
		observationSize += result.Size()
		if observationSize > sizeLimit {
			break
		}
		obs.Performable = append(obs.Performable, result)
		added++
	}

	hook.logger.Printf("adding %d results to observation", added)

	return nil
}
