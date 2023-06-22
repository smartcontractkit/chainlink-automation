package preprocessors

import (
	"context"
	"log"

	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg"
)

type LogEventProvider interface {
	// GetLogs returns the latest logs
	GetLogs(context.Context) ([]ocr2keepers.UpkeepPayload, error)
}

type logPreProcessor struct {
	lggr     *log.Logger
	provider LogEventProvider
}

func NewLogPreProcessor(lggr *log.Logger, provider LogEventProvider) *logPreProcessor {
	return &logPreProcessor{
		lggr:     lggr,
		provider: provider,
	}
}

func (p *logPreProcessor) PreProcess(ctx context.Context, _ []ocr2keepers.UpkeepPayload) ([]ocr2keepers.UpkeepPayload, error) {
	payloads, err := p.provider.GetLogs(ctx)
	if err != nil {
		return nil, err
	}
	p.lggr.Printf("received %d payloads from log event provider", len(payloads))
	// TODO: filter out payloads that were already processed
	// 		 by a quorum of node in the previous observation
	return payloads, err
}
