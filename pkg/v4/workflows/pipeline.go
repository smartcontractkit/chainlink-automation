package workflows

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/smartcontractkit/chainlink-automation/internal/util"
	ocr2keepers "github.com/smartcontractkit/chainlink-automation/pkg/v3"
	"github.com/smartcontractkit/chainlink-automation/pkg/v3/postprocessors"
	"github.com/smartcontractkit/chainlink-common/pkg/types/automation"
)

const (
	observationProcessLimit = 20 * time.Second
)

type WorkflowProvider interface {
	GetPayloads(ctx context.Context) ([]automation.UpkeepPayload, error)
	GetPreprocessors() []ocr2keepers.PreProcessor
	GetPostprocessor() postprocessors.PostProcessor
	GetRunner() ocr2keepers.Runner
}

type Pipeline struct {
	closer           util.Closer
	logger           *log.Logger
	interval         time.Duration
	workflowProvider WorkflowProvider
}

func NewPipeline(provider WorkflowProvider, interval time.Duration, logger *log.Logger) *Pipeline {
	return &Pipeline{
		logger:           logger,
		interval:         interval,
		workflowProvider: provider,
	}
}

func (p *Pipeline) Start(pctx context.Context) error {
	ctx, cancel := context.WithCancel(pctx)
	defer cancel()

	if !p.closer.Store(cancel) {
		return fmt.Errorf("already running")
	}

	p.logger.Printf("starting ticker service")
	defer p.logger.Printf("ticker service stopped")

	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			payloads, err := p.workflowProvider.GetPayloads(ctx)
			if err != nil {
				p.logger.Printf("error fetching payloads: %s", err.Error())
			}

			preprocessor := p.workflowProvider.GetPreprocessors()

			postprocessor := p.workflowProvider.GetPostprocessor()

			runner := p.workflowProvider.GetRunner()

			// Process can be a heavy call taking upto ObservationProcessLimit seconds
			// so it is run in a separate goroutine to not block further ticks
			// Exploratory: Add some control to limit the number of goroutines spawned
			go func(ctx context.Context, payloads []automation.UpkeepPayload, l *log.Logger) {
				if err := p.Process(ctx, p.logger, payloads, preprocessor, postprocessor, runner); err != nil {
					l.Printf("error processing observer: %s", err.Error())
				}
			}(ctx, payloads, p.logger)
		case <-ctx.Done():
			return nil
		}
	}
}

func (p *Pipeline) Process(ctx context.Context, logger *log.Logger, payloads []automation.UpkeepPayload, preprocessors []ocr2keepers.PreProcessor, postprocessor postprocessors.PostProcessor, runner ocr2keepers.Runner) error {
	pCtx, cancel := context.WithTimeout(ctx, observationProcessLimit)
	defer cancel()

	logger.Printf("got %d payloads from ticker", len(payloads))

	var err error

	// Run pre-processors
	for _, preprocessor := range preprocessors {
		payloads, err = preprocessor.PreProcess(pCtx, payloads)
		if err != nil {
			return err
		}
	}

	logger.Printf("processing %d payloads", len(payloads))

	// Run check pipeline
	results, err := runner.CheckUpkeeps(pCtx, payloads...)
	if err != nil {
		return err
	}

	logger.Printf("post-processing %d results", len(results))

	// Run post-processor
	if err := postprocessor.PostProcess(pCtx, results, payloads); err != nil {
		return err
	}

	logger.Printf("finished processing of %d results: %+v", len(results), results)

	return nil
}

func (p *Pipeline) Close() error {
	_ = p.closer.Close()
	return nil
}
