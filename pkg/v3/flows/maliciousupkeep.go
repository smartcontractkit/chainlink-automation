package flows

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/big"
	"time"

	common "github.com/smartcontractkit/chainlink-common/pkg/types/automation"

	ocr2keepersv3 "github.com/smartcontractkit/chainlink-automation/pkg/v3"
	"github.com/smartcontractkit/chainlink-automation/pkg/v3/service"
	"github.com/smartcontractkit/chainlink-automation/pkg/v3/telemetry"
	"github.com/smartcontractkit/chainlink-automation/pkg/v3/tickers"
	"github.com/smartcontractkit/chainlink-automation/pkg/v3/types"
)

// newMaliciousUpkeepFlow is the happy path entry point for log triggered upkeeps
func newMaliciousUpkeepFlow(
	ms types.MetadataStore,
	mup common.MaliciousUpkeepProvider,
	maliciousUpkeepReportingInterval time.Duration,
	logger *log.Logger,
) service.Recoverable {
	obs := ocr2keepersv3.NewUpkeepIdsObserver()

	timeTick := tickers.NewTimeTicker[[]*big.Int](maliciousUpkeepReportingInterval, obs, func(ctx context.Context, _ time.Time) (tickers.Tick[[]*big.Int], error) {
		return maliciousUpkeepTick{logger: logger, ms: ms, mup: mup}, nil
	}, log.New(logger.Writer(), fmt.Sprintf("[%s | malicious-upkeep-ticker]", telemetry.ServiceName), telemetry.LogPkgStdFlags))

	return timeTick
}

type maliciousUpkeepTick struct {
	mup    common.MaliciousUpkeepProvider
	ms     types.MetadataStore
	logger *log.Logger
}

func (et maliciousUpkeepTick) Value(ctx context.Context) ([]*big.Int, error) {
	if et.mup == nil {
		return nil, nil
	}

	ids, err := et.mup.GetMaliciousUpkeepIds(ctx)
	if err != nil {
		return nil, err
	}

	// save it to metadata_store
	et.ms.SetMaliciousUpkeeps(ids)

	et.logger.Printf("%d malicious upkeeps returned by malicious upkeep provider. they are %s", len(ids), ids)

	return nil, errors.New("no-op")
}
