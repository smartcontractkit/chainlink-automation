package flows

// import (
// 	"context"
// 	"io"
// 	"log"
// 	"sync"
// 	"testing"
// 	"time"

// 	"github.com/stretchr/testify/assert"
// 	"github.com/stretchr/testify/mock"

// 	"github.com/smartcontractkit/ocr2keepers/pkg/util"
// 	ocr2keepersv3 "github.com/smartcontractkit/ocr2keepers/pkg/v3"
// 	"github.com/smartcontractkit/ocr2keepers/pkg/v3/flows/mocks"
// 	"github.com/smartcontractkit/ocr2keepers/pkg/v3/service"
// 	"github.com/smartcontractkit/ocr2keepers/pkg/v3/store"
// 	"github.com/smartcontractkit/ocr2keepers/pkg/v3/tickers"
// 	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg/v3/types"
// 	mocks2 "github.com/smartcontractkit/ocr2keepers/pkg/v3/types/mocks"
// )

import (
	"context"
	"io"
	"log"
	"sync"
	"testing"
	"time"

	ocr2keepersv3 "github.com/smartcontractkit/ocr2keepers/pkg/v3"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/flows/mocks"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/retryqueue"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/service"
	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg/v3/types"
	ocr2keepersmocks "github.com/smartcontractkit/ocr2keepers/pkg/v3/types/mocks"
	typesmocks "github.com/smartcontractkit/ocr2keepers/pkg/v3/types/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestLogTriggerFlow(t *testing.T) {
	logger := log.New(io.Discard, "", log.LstdFlags)

	times := 2

	runner := new(mocks.MockRunner)
	rStore := new(mocks.MockResultStore)
	coord := new(ocr2keepersmocks.MockCoordinator)
	retryQ := retryqueue.NewRetryQueue(logger)
	upkeepStateUpdater := new(ocr2keepersmocks.MockUpkeepStateUpdater)
	lp := new(typesmocks.MockLogEventProvider)

	lp.On("GetLatestPayloads", mock.Anything).Return([]ocr2keepers.UpkeepPayload{
		{
			UpkeepID: ocr2keepers.UpkeepIdentifier([32]byte{1}),
			WorkID:   "0x1",
		},
		{
			UpkeepID: ocr2keepers.UpkeepIdentifier([32]byte{2}),
			WorkID:   "0x2",
		},
	}, nil).Times(times)
	coord.On("PreProcess", mock.Anything, mock.Anything).Return([]ocr2keepers.UpkeepPayload{
		{
			UpkeepID: ocr2keepers.UpkeepIdentifier([32]byte{1}),
			WorkID:   "0x1",
		},
		{
			UpkeepID: ocr2keepers.UpkeepIdentifier([32]byte{2}),
			WorkID:   "0x2",
		},
	}, nil).Times(times)
	runner.On("CheckUpkeeps", mock.Anything, mock.Anything, mock.Anything).Return([]ocr2keepers.CheckResult{
		{
			UpkeepID: ocr2keepers.UpkeepIdentifier([32]byte{1}),
			WorkID:   "0x1",
			Eligible: true,
		},
		{
			UpkeepID:  ocr2keepers.UpkeepIdentifier([32]byte{2}),
			WorkID:    "0x2",
			Retryable: true,
		},
	}, nil).Times(times)
	// within the 3 ticks, it should retry twice and the third time it should be eligible and add to result store
	rStore.On("Add", mock.Anything).Times(times)

	// set the ticker time lower to reduce the test time
	logInterval := 50 * time.Millisecond

	svc := newLogTriggerFlow([]ocr2keepersv3.PreProcessor[ocr2keepers.UpkeepPayload]{coord},
		rStore, runner, lp, logInterval, retryQ, upkeepStateUpdater, logger)

	var wg sync.WaitGroup
	wg.Add(1)

	go func(svc service.Recoverable, ctx context.Context) {
		assert.NoError(t, svc.Start(ctx))
		wg.Done()
	}(svc, context.Background())

	time.Sleep(logInterval*time.Duration(times) + logInterval/2)

	assert.NoError(t, svc.Close(), "no error expected on shut down")

	rStore.AssertExpectations(t)

	wg.Wait()
}
