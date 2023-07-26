package plugin

import (
	"fmt"

	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg"
	ocr2keepersv3 "github.com/smartcontractkit/ocr2keepers/pkg/v3"
)

type recoverables struct {
	threshold   int
	resultCount map[string]resultAndCount[ocr2keepers.CoordinatedProposal]
}

func newRecoverables(threshold int) *recoverables {
	return &recoverables{
		threshold:   threshold,
		resultCount: make(map[string]resultAndCount[ocr2keepers.CoordinatedProposal]),
	}
}

func (r *recoverables) add(observation ocr2keepersv3.AutomationObservation) {
	raw, ok := observation.Metadata[ocr2keepersv3.RecoveryProposalObservationKey]
	if !ok {
		return
	}

	proposals, ok := raw.([]ocr2keepers.CoordinatedProposal)
	if !ok {
		return
	}

	for _, proposal := range proposals {
		key := fmt.Sprintf("%v", proposal)

		count, ok := r.resultCount[key]

		if !ok {
			count = resultAndCount[ocr2keepers.CoordinatedProposal]{
				result: proposal,
				count:  1,
			}
		} else {
			count.count++
		}

		r.resultCount[key] = count
	}
}

func (r *recoverables) set(outcome *ocr2keepersv3.AutomationOutcome) {
	var recoverable []ocr2keepers.CoordinatedProposal

	for _, rec := range r.resultCount {
		if rec.count >= r.threshold {
			recoverable = append(recoverable, rec.result)
		}
	}

	outcome.Metadata[ocr2keepersv3.CoordinatedRecoveryProposalKey] = recoverable
}
