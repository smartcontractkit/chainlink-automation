package postprocessors

import (
	"context"

	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg"
)

type checkResultRecoverer interface {
	// Recover submits a check result for retry
	Recover(ocr2keepers.CheckResult) error
}

type recoveryPostProcessor struct {
	recoverer checkResultRecoverer
}

func (p *recoveryPostProcessor) PostProcess(_ context.Context, results []ocr2keepers.CheckResult) error {
	for _, res := range results {
		if res.Retryable {
			if err := p.recoverer.Recover(res); err != nil {
				return err
			}
		}
	}
	return nil
}

func NewRecoveryPostProcessor(recoverer checkResultRecoverer) *recoveryPostProcessor {
	return &recoveryPostProcessor{
		recoverer: recoverer,
	}
}
