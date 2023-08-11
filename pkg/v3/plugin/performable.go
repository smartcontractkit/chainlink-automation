package plugin

import (
	"math/rand"

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
	resultCount   map[string]resultAndCount[ocr2keepers.CheckResult]
}

// Performables gets quorum on agreed check results which should ultimately be
// performed within a report. It assumes only valid observations are added to it
// and simply adds all results which achieve the threshold quorum.
// Results are agreed upon by their UniqueID() which contains all the data
// to be sent to the contract, ensuring that the whole result gets quorum
func newPerformables(threshold int, limit int, rSrc [16]byte) *performables {
	return &performables{
		threshold:   threshold,
		resultCount: make(map[string]resultAndCount[ocr2keepers.CheckResult]),
	}
}

func (p *performables) add(observation ocr2keepersv3.AutomationObservation) {
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
}

func (p *performables) set(outcome *ocr2keepersv3.AutomationOutcome) {
	var performable []ocr2keepers.CheckResult

	for _, payload := range p.resultCount {
		if payload.count > p.threshold {
			performable = append(performable, payload.result)
		}
	}

	rand.New(util.NewKeyedCryptoRandSource(p.keyRandSource)).Shuffle(len(performable), func(i, j int) {
		performable[i], performable[j] = performable[j], performable[i]
	})

	if len(performable) > p.limit {
		performable = performable[:p.limit]
	}
	outcome.AgreedPerformables = performable
}
