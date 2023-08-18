package plugin

import (
	"log"
	"sort"

	"github.com/smartcontractkit/ocr2keepers/internal/util"
	ocr2keepersv3 "github.com/smartcontractkit/ocr2keepers/pkg/v3"
	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg/v3/types"
)

type resultAndCount[T any] struct {
	result T
	count  int
}

type performables struct {
	limit         int
	keyRandSource [16]byte
	threshold     int
	logger        *log.Logger
	resultCount   map[string]resultAndCount[ocr2keepers.CheckResult]
}

// Performables gets quorum on agreed check results which should ultimately be
// performed within a report. It assumes only valid observations are added to it
// and simply adds all results which achieve the threshold quorum.
// Results are agreed upon by their UniqueID() which contains all the data
// withn the result.
func newPerformables(threshold int, limit int, rSrc [16]byte, logger *log.Logger) *performables {
	return &performables{
		threshold:     threshold,
		limit:         limit,
		keyRandSource: rSrc,
		logger:        logger,
		resultCount:   make(map[string]resultAndCount[ocr2keepers.CheckResult]),
	}
}

func (p *performables) add(observation ocr2keepersv3.AutomationObservation) {
	initialCount := len(p.resultCount)
	for _, result := range observation.Performable {
		uid := result.UniqueID()
		payloadCount, ok := p.resultCount[uid]
		if !ok {
			payloadCount = resultAndCount[ocr2keepers.CheckResult]{
				result: result,
				count:  1,
			}
		} else {
			payloadCount.count++
		}

		p.resultCount[uid] = payloadCount
	}
	p.logger.Printf("Added %d new results from %d performables", len(p.resultCount)-initialCount, len(observation.Performable))
}

func (p *performables) set(outcome *ocr2keepersv3.AutomationOutcome) {
	performable := make([]ocr2keepers.CheckResult, 0)

	added := 0
	for _, payload := range p.resultCount {
		if payload.count > p.threshold {
			added++
			performable = append(performable, payload.result)
		}
	}
	p.logger.Printf("Adding %d agreed performables over threshold %d", added, p.threshold)

	// Sort by a shuffled workID.
	sort.Slice(performable, func(i, j int) bool {
		return util.ShuffleString(performable[i].WorkID, p.keyRandSource) < util.ShuffleString(performable[j].WorkID, p.keyRandSource)
	})

	if len(performable) > p.limit {
		p.logger.Printf("Limiting new performables in outcome to %d", p.limit)
		performable = performable[:p.limit]
	}
	p.logger.Printf("Setting outcome.AgreedPerformables with %d performables", len(performable))
	outcome.AgreedPerformables = performable
}
