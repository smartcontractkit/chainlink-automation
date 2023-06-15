package postprocessors

import (
	"context"

	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg"
)

type Recoverer interface {
	// Recover submits a check result for retry
	Recover(ocr2keepers.CheckResult) error
}

type recoveryPostProcessor struct {
	recoverer Recoverer
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

func NewRecoveryPostProcessor(recoverer Recoverer) *recoveryPostProcessor {
	return &recoveryPostProcessor{
		recoverer: recoverer,
	}
}
