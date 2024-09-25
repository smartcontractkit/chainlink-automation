package log

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/smartcontractkit/chainlink-automation/pkg/v3/types"

	ocr2keepersv3 "github.com/smartcontractkit/chainlink-automation/pkg/v3"
	"github.com/smartcontractkit/chainlink-automation/pkg/v3/postprocessors"
	"github.com/smartcontractkit/chainlink-automation/pkg/v3/preprocessors"
	"github.com/smartcontractkit/chainlink-automation/pkg/v3/service"
	"github.com/smartcontractkit/chainlink-automation/pkg/v3/telemetry"
	"github.com/smartcontractkit/chainlink-automation/pkg/v3/tickers"
	common "github.com/smartcontractkit/chainlink-common/pkg/types/automation"
)

const (
	// This is the ticker interval for recovery proposal flow
	recoveryProposalInterval = 1 * time.Second
)

func NewRecoveryProposalFlow(
	preProcessors []ocr2keepersv3.PreProcessor,
	runner ocr2keepersv3.Runner,
	metadataStore types.MetadataStore,
	recoverableProvider common.RecoverableProvider,
	stateUpdater common.UpkeepStateUpdater,
	logger *log.Logger,
) service.Recoverable {
	preProcessors = append(preProcessors, preprocessors.NewProposalFilterer(metadataStore, types.LogTrigger))
	postprocessors := postprocessors.NewCombinedPostprocessor(
		postprocessors.NewIneligiblePostProcessor(stateUpdater, logger),
		postprocessors.NewAddProposalToMetadataStorePostprocessor(metadataStore),
	)

	observerLggrPrefix := fmt.Sprintf("[%s | recovery-proposal-observer]", telemetry.ServiceName)
	observerLggr := log.New(logger.Writer(), observerLggrPrefix, telemetry.LogPkgStdFlags)

	observer := ocr2keepersv3.NewRunnableObserver(
		preProcessors,
		postprocessors,
		runner,
		observationProcessLimit,
		observerLggr,
	)

	getterFn := func(ctx context.Context, _ time.Time) (tickers.Tick, error) {
		return logRecoveryTick{logger: logger, logRecoverer: recoverableProvider}, nil
	}

	lggrPrefix := fmt.Sprintf("[%s | recovery-proposal-ticker]", telemetry.ServiceName)
	lggr := log.New(logger.Writer(), lggrPrefix, telemetry.LogPkgStdFlags)

	return tickers.NewTimeTicker(recoveryProposalInterval, observer, getterFn, lggr)
}

type logRecoveryTick struct {
	logRecoverer common.RecoverableProvider
	logger       *log.Logger
}

func (t logRecoveryTick) Value(ctx context.Context) ([]common.UpkeepPayload, error) {
	if t.logRecoverer == nil {
		return nil, nil
	}

	logs, err := t.logRecoverer.GetRecoveryProposals(ctx)

	t.logger.Printf("%d logs returned by log recoverer", len(logs))

	return logs, err
}
