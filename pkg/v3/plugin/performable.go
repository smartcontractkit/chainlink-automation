package plugin

import (
	"encoding/json"
	"fmt"

	ocr2keepersv3 "github.com/smartcontractkit/ocr2keepers/pkg/v3"
	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg/v3/types"
)

type resultAndCount[T any] struct {
	result T
	count  int
}

type performables struct {
	threshold   int
	resultCount map[string]resultAndCount[ocr2keepers.CheckResult]
}

func newPerformables(threshold int) *performables {
	return &performables{
		threshold:   threshold,
		resultCount: make(map[string]resultAndCount[ocr2keepers.CheckResult]),
	}
}

func (p *performables) add(observation ocr2keepersv3.AutomationObservation) {
	for _, result := range observation.Performable {
		if !result.Eligible {
			continue
		}

		fmt.Printf("%+v", result)
		// TODO: Handle error here, or probably find a better solution than json marshalling
		resultBytes, _ := json.Marshal(result)
		uid := fmt.Sprintf("%v", resultBytes)
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

	outcome.Performable = performable
}
