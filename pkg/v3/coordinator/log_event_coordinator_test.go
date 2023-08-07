package coordinator

import (
	"context"
	"io"
	"log"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/smartcontractkit/ocr2keepers/pkg/util"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/coordinator/mocks"
	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg/v3/types"
)

func TestLogEventCoordinator(t *testing.T) {
	setup := func(t *testing.T, logger *log.Logger) (*reportCoordinator, *mocks.EventProvider) {
		logs := new(mocks.EventProvider)

		return &reportCoordinator{
			logger:            logger,
			events:            logs,
			activeKeys:        util.NewCache[bool](time.Hour), // 1 hour allows the cleanup routine to clear stale data
			activeKeysCleaner: util.NewIntervalCacheCleaner[bool](DefaultCacheClean),
			minConfs:          1,
			chStop:            make(chan struct{}, 1),
		}, logs
	}

	// TODO (auto-4183 Finalize coordinator)
	// t.Run("Accept", func(t *testing.T) {
	// 	rc, _ := setup(t, log.New(io.Discard, "nil", 0))
	// 	upkeep := ocr2keepers.ReportedUpkeep{
	// 		ID: "your-upkeep-id",
	// 		Trigger: ocr2keepers.Trigger{
	// 			Extension: map[string]interface{}{
	// 				"txHash": "your-tx-hash",
	// 			},
	// 		},
	// 	}

	// 	assert.NoError(t, rc.Accept(upkeep), "no error expected from accepting the key")
	// 	assert.NoError(t, rc.Accept(upkeep), "Key can get accepted again")
	// })

	t.Run("Check Event", func(t *testing.T) {
		rc, logs := setup(t, log.New(io.Discard, "nil", 0))
		ctx := context.Background()
		expectedEvents := []ocr2keepers.TransmitEvent{
			{Type: ocr2keepers.PerformEvent, Confirmations: 3},
			{Type: ocr2keepers.StaleReportEvent, Confirmations: 2},
			{Type: ocr2keepers.ReorgReportEvent, Confirmations: 4},
		}

		logs.On("Events", mock.Anything).Return(expectedEvents, nil).Once()

		err := rc.checkEvents(ctx)

		logs.AssertExpectations(t)
		assert.NoError(t, err, "expected no error from checking events")
	})

	t.Run("Perform Event", func(t *testing.T) {
		rc, _ := setup(t, log.New(io.Discard, "nil", 0))
		evt := ocr2keepers.TransmitEvent{
			ID: "your-event-id",
		}

		rc.performEvent(evt)

		value, ok := rc.activeKeys.Get(evt.ID)
		assert.True(t, ok, "expected active key to exist")
		assert.Equal(t, true, value, "expected active key value to be true")
	})

	t.Run("Pre Process", func(t *testing.T) {
		rc, _ := setup(t, log.New(io.Discard, "nil", 0))
		rc.activeKeys.Set("payload1", true, util.DefaultCacheExpiration)

		// Define some example payloads
		payloads := []ocr2keepers.UpkeepPayload{
			{ID: "payload1"},
			{ID: "payload2"},
			{ID: "payload3"},
		}

		// Call the PreProcess git st
		filteredPayloads, err := rc.PreProcess(context.Background(), payloads)
		assert.NoError(t, err, "no error expected from PreProcess")

		// Assert that only payload2 and payload3 are included in the filteredPayloads slice
		expectedPayloads := []ocr2keepers.UpkeepPayload{
			{ID: "payload2"},
			{ID: "payload3"},
		}
		assert.Equal(t, expectedPayloads, filteredPayloads, "filteredPayloads should match the expected payloads")
	})
}
