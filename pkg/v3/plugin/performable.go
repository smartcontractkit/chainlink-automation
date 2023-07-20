package plugin

import (
	"fmt"

	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg"
	ocr2keepersv3 "github.com/smartcontractkit/ocr2keepers/pkg/v3"
)

type resultAndCount struct {
	result ocr2keepers.CheckResult
	count  int
}

type performables struct {
	threshold   int
	resultCount map[string]resultAndCount
}

func newPerformables(threshold int) *performables {
	return &performables{
		threshold:   threshold,
		resultCount: make(map[string]resultAndCount),
	}
}

func (p *performables) add(observation ocr2keepersv3.AutomationObservation) {
	for _, result := range observation.Performable {
		uid := fmt.Sprintf("%v", result)
		payloadCount, ok := p.resultCount[uid]

		if !ok {
			payloadCount = resultAndCount{
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