package flows

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"time"

	common "github.com/smartcontractkit/chainlink-common/pkg/types/automation"

	ocr2keepersv3 "github.com/smartcontractkit/chainlink-automation/pkg/v3"
	"github.com/smartcontractkit/chainlink-automation/pkg/v3/postprocessors"
	"github.com/smartcontractkit/chainlink-automation/pkg/v3/service"
	"github.com/smartcontractkit/chainlink-automation/pkg/v3/telemetry"
	"github.com/smartcontractkit/chainlink-automation/pkg/v3/tickers"
	"github.com/smartcontractkit/chainlink-automation/pkg/v3/types"
)

// log trigger flow is the happy path entry point for log triggered upkeeps
func newMaliciousUpkeepFlow(
	preprocessors []ocr2keepersv3.PreProcessor[*big.Int],
	rs types.ResultStore,
	rn ocr2keepersv3.Runner,
	mup common.MaliciousUpkeepProvider,
	maliciousUpkeepReportingInterval time.Duration,
	logger *log.Logger,
) service.Recoverable {
	post := postprocessors.NewCombinedPostprocessor(
		postprocessors.NewEligiblePostProcessor(rs, telemetry.WrapLogger(logger, "log-trigger-eligible-postprocessor")),
	)

	obs := ocr2keepersv3.NewRunnableObserver(
		preprocessors,
		post,
		rn,
		ObservationProcessLimit,
		log.New(logger.Writer(), fmt.Sprintf("[%s | log-trigger-observer]", telemetry.ServiceName), telemetry.LogPkgStdFlags),
	)

	timeTick := tickers.NewTimeTicker[[]*big.Int](maliciousUpkeepReportingInterval, obs, func(ctx context.Context, _ time.Time) (tickers.Tick[[]*big.Int], error) {
		return maliciousUpkeepTick{logger: logger, mup: mup}, nil
	}, log.New(logger.Writer(), fmt.Sprintf("[%s | log-trigger-ticker]", telemetry.ServiceName), telemetry.LogPkgStdFlags))

	return timeTick
}

type maliciousUpkeepTick struct {
	mup    common.MaliciousUpkeepProvider
	logger *log.Logger
}

func (et maliciousUpkeepTick) Value(ctx context.Context) ([]*big.Int, error) {
	if et.mup == nil {
		return nil, nil
	}

	logs, err := et.mup.GetMaliciousUpkeepIds(ctx)

	et.logger.Printf("%d logs returned by log provider", len(logs))

	return logs, err
}
