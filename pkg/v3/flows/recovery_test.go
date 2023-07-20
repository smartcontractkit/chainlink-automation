package flows

import (
	"context"
	"io"
	"log"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg"
	ocr2keepersv3 "github.com/smartcontractkit/ocr2keepers/pkg/v3"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/flows/mocks"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/service"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/store"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/tickers"
)

func TestRecoveryFlow(t *testing.T) {
	logger := log.New(io.Discard, "", log.LstdFlags)

	mStore := new(mocks.MockMetadataStore)
	rec := new(mocks.MockRecoverableProvider)
	configFuncs := []tickers.ScheduleTickerConfigFunc{ // retry configs
		tickers.ScheduleTickerWithDefaults,
		func(c *tickers.ScheduleTickerConfig) {
			c.SendDelay = 30 * time.Millisecond
		},
	}
	// preprocessor is just a pass through
	coord := new(mockedPreprocessor)
	testData := []ocr2keepers.UpkeepPayload{
		{ID: "test"},
	}
	preprocessors := []ocr2keepersv3.PreProcessor[ocr2keepers.UpkeepPayload]{coord}

	rec.On("GetRecoverables").Return(testData, nil).Times(1)
	rec.On("GetRecoverables").Return(nil, nil).Times(3)

	// metadata store should set the value
	mStore.On("Set", store.ProposalRecoveryMetadata, testData).Times(1)
	mStore.On("Set", store.ProposalRecoveryMetadata, []ocr2keepers.UpkeepPayload{}).Times(3)

	// set the ticker time lower to reduce the test time
	recoveryInterval := 50 * time.Millisecond

	svc, _ := newRecoveryProposalFlow(preprocessors, mStore, rec, recoveryInterval, logger, configFuncs...)

	var wg sync.WaitGroup
	wg.Add(1)

	go func(svc service.Recoverable, ctx context.Context) {
		assert.NoError(t, svc.Start(ctx))
		wg.Done()
	}(svc, context.Background())

	time.Sleep(210 * time.Millisecond)

	assert.NoError(t, svc.Close(), "no error expected on shut down")

	assert.Equal(t, 4, coord.Calls())
	mStore.AssertExpectations(t)

	wg.Wait()
}