package keepers

import (
	"io"
	"log"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/smartcontractkit/ocr2keepers/pkg/types"
)

func TestReportCoordinator(t *testing.T) {
	mr := types.NewMockRegistry(t)
	mp := types.NewMockPerformLogProvider(t)

	l := log.New(io.Discard, "nil", 0)

	rc := &reportCoordinator{
		logger:     l,
		registry:   mr,
		logs:       mp,
		idBlocks:   newCache[types.BlockKey](time.Second),
		activeKeys: newCache[bool](time.Minute),
		minConfs:   1,
		chStop:     make(chan struct{}),
	}

	// set up the mocks and mock data
	key1Block1 := types.UpkeepKey("1|1")
	key1Block2 := types.UpkeepKey("2|1")
	id1 := types.UpkeepIdentifier("1")
	bk2 := types.BlockKey("2")
	filter := rc.Filter()

	t.Run("FilterBeforeAccept", func(t *testing.T) {
		// calling filter at this point should return true because the key has not
		// yet been added to the filter
		mr.Mock.On("IdentifierFromKey", key1Block1).Return(id1, nil)
		assert.Equal(t, true, filter(key1Block1), "should not filter out key 1 at block 1: key has not been accepted")

		mr.Mock.On("IdentifierFromKey", key1Block2).Return(id1, nil)
		assert.Equal(t, true, filter(key1Block2), "should not filter out key 1 at block 2: key has not been accepted")

		// is transmission confirmed should also return true because the key has
		// not been set in the filter
		// this would block any transmissions for this upkeep key (block & id)
		assert.Equal(t, true, rc.IsTransmissionConfirmed(key1Block1), "transmit will return confirmed: key has not been accepted")
	})

	t.Run("Accept", func(t *testing.T) {
		mr.Mock.On("IdentifierFromKey", key1Block1).Return(id1, nil)
		assert.NoError(t, rc.Accept(key1Block1), "no error expected from accepting the key")
		assert.ErrorIs(t, rc.Accept(key1Block1), ErrKeyAlreadySet, "key should not be accepted again and should return an error")
	})

	t.Run("FilterAfterAccept", func(t *testing.T) {
		// no logs have been read at this point so the upkeep key should be filtered
		// out at all block numbers
		mr.Mock.On("IdentifierFromKey", key1Block1).Return(id1, nil)
		assert.Equal(t, false, filter(key1Block1), "filter should return false to indicate key should be filtered out at block 1")

		mr.Mock.On("IdentifierFromKey", key1Block2).Return(id1, nil)
		assert.Equal(t, false, filter(key1Block2), "filter should return false to indicate key should be filtered out at block 2")

		assert.Equal(t, false, rc.IsTransmissionConfirmed(key1Block1), "transmit should not be confirmed: key is now set, but no logs have been identified")

		// returning true for an unset key prevents any node from transmitting a key that was never accepted
		assert.Equal(t, true, rc.IsTransmissionConfirmed(key1Block2), "transmit should be confirmed: key was not set for block 2")
	})

	t.Run("CollectLogsWithMinConfirmations_LessThan", func(t *testing.T) {
		mr.Mock.On("IdentifierFromKey", key1Block1).Return(id1, nil)
		mp.Mock.On("PerformLogs", mock.Anything).Return([]types.PerformLog{
			{Key: key1Block1, TransmitBlock: bk2, Confirmations: 0},
		}, nil).Once()

		rc.checkLogs()

		// perform log didn't have the threshold number of confirmations
		// making the key still locked at all blocks
		mr.Mock.On("IdentifierFromKey", key1Block1).Return(id1, nil)
		assert.Equal(t, false, filter(key1Block1), "filter should return false to indicate key should be filtered out at block 1")

		mr.Mock.On("IdentifierFromKey", key1Block2).Return(id1, nil)
		assert.Equal(t, false, filter(key1Block2), "filter should return false to indicate key should be filtered out at block 2")

		assert.Equal(t, false, rc.IsTransmissionConfirmed(key1Block1), "transmit should not be confirmed: the key is now set, but no logs have been identified")
		assert.Equal(t, true, rc.IsTransmissionConfirmed(key1Block2), "transmit should be confirmed: key was not set for block 2")
	})

	t.Run("CollectLogsWithMinConfirmations_GreaterThan", func(t *testing.T) {
		mr.Mock.On("IdentifierFromKey", key1Block1).Return(id1, nil)
		mp.Mock.On("PerformLogs", mock.Anything).Return([]types.PerformLog{
			{Key: key1Block1, TransmitBlock: bk2, Confirmations: 1},
		}, nil).Once()

		rc.checkLogs()

		// because the transmit block is block 2, the filter should continue
		// to filter out key up to block 2
		mr.Mock.On("IdentifierFromKey", key1Block1).Return(id1, nil)
		assert.Equal(t, false, filter(key1Block1), "filter should return false to indicate key should be filtered out at block 1")

		mr.Mock.On("IdentifierFromKey", key1Block2).Return(id1, nil)
		assert.Equal(t, true, filter(key1Block2), "filter should return true to indicate key should not be filtered out at block 2")

		assert.Equal(t, true, rc.IsTransmissionConfirmed(key1Block1), "transmit should be confirmed")
		assert.Equal(t, true, rc.IsTransmissionConfirmed(key1Block2), "transmit should be confirmed: key was not set for block 2")

		assert.ErrorIs(t, rc.Accept(key1Block1), ErrKeyAlreadySet, "key should not be accepted after transmit confirmed and should return an error")
	})

	mp.AssertExpectations(t)
	mr.AssertExpectations(t)
}
