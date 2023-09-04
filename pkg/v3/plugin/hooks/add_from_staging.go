package hooks

import (
	"fmt"
	"log"
	"sort"

	ocr2keepersv3 "github.com/smartcontractkit/ocr2keepers/pkg/v3"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/random"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/telemetry"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/types"
	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg/v3/types"
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

func (hook *AddFromStagingHook) RunHook(obs *ocr2keepersv3.AutomationObservation, limit int, rSrc [16]byte) error {
	results, err := hook.store.View()
	if err != nil {
		return err
	}
	results, err = hook.coord.FilterResults(results)
	if err != nil {
		return err
	}

	// Sort by a shuffled workID. workID for all items is shuffled using a pseduorandom souce
	// that is the same across all nodes for a given round. This ensures that all nodes try to
	// send the same subset of workIDs if they are available, while giving different priority
	// to workIDs in different rounds.
	sort.Slice(results, func(i, j int) bool {
		return random.ShuffleString(results[i].WorkID, rSrc) < random.ShuffleString(results[j].WorkID, rSrc)
	})
	if len(results) > limit {
		results = results[:limit]
	}

	hook.logger.Printf("adding %d results to observation", len(results))
	obs.Performable = append(obs.Performable, results...)

	return nil
}
