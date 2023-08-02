package preprocessors

import (
	"context"

	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg"
)

type mercuryPreprocessor struct {
	mercuryLookup bool
}

// NewMercuryPreprocessor returns an instance of the mercury preprocessor
func NewMercuryPreprocessor(mercuryLookup bool) *mercuryPreprocessor {
	return &mercuryPreprocessor{
		mercuryLookup: mercuryLookup,
	}
}

func (p *mercuryPreprocessor) PreProcess(_ context.Context, payloads []ocr2keepers.UpkeepPayload) ([]ocr2keepers.UpkeepPayload, error) {
	var filteredPayloads []ocr2keepers.UpkeepPayload

	for _, payload := range payloads {
		if p.mercuryLookup {
			payload.EnableMercuryLookup()
		}
		filteredPayloads = append(filteredPayloads, payload)
	}

	return filteredPayloads, nil
}
