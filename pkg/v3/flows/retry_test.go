package flows

import (
	"context"
	"io"
	"log"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	ocr2keepersv3 "github.com/smartcontractkit/ocr2keepers/pkg/v3"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/flows/mocks"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/service"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/tickers"
	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg/v3/types"
)

func TestRetryFlow(t *testing.T) {
	logger := log.New(io.Discard, "", log.LstdFlags)

	runner := &mockedRunner{eligibleAfter: 2}
	rStore := new(mocks.MockResultStore)
	recoverer := new(mocks.MockRetryer)
	configFuncs := []tickers.ScheduleTickerConfigFunc{ // retry configs
		tickers.ScheduleTickerWithDefaults,
		func(c *tickers.ScheduleTickerConfig) {
			c.SendDelay = 30 * time.Millisecond
			// set the max send duration high to avoid retry failure
			c.MaxSendDuration = 1000 * time.Millisecond
		},
	}
	coord := new(mockedPreprocessor)
	preprocessors := []ocr2keepersv3.PreProcessor[ocr2keepers.UpkeepPayload]{coord}

	// within the 3 ticks, it should retry twice and the third time it should be eligible and add to result store
	rStore.On("Add", mock.Anything).Times(1)

	// set the ticker time lower to reduce the test time
	retryInterval := 50 * time.Millisecond

	svc, retryer := newRetryFlow(preprocessors, rStore, runner, recoverer, retryInterval, logger, configFuncs...)

	testCheckResult := ocr2keepers.CheckResult{
		Retryable: true,
		UpkeepID:  ocr2keepers.UpkeepIdentifier([32]byte{1}),
	}

	var wg sync.WaitGroup
	wg.Add(1)

	go func(svc service.Recoverable, ctx context.Context) {
		assert.NoError(t, svc.Start(ctx))
		wg.Done()
	}(svc, context.Background())

	if err := retryer.Retry(testCheckResult); err != nil {
		return
	}

	time.Sleep(160 * time.Millisecond)

	assert.NoError(t, svc.Close(), "no error expected on shut down")

	assert.Equal(t, 3, coord.Calls())
	rStore.AssertExpectations(t)

	wg.Wait()
}
