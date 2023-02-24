package keepers

import (
	"io"
	"log"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/smartcontractkit/ocr2keepers/pkg/chain"
	"github.com/smartcontractkit/ocr2keepers/pkg/types"
	pkgutil "github.com/smartcontractkit/ocr2keepers/pkg/util"
)

func TestReportCoordinator(t *testing.T) {
	setup := func(t *testing.T, l *log.Logger) (*reportCoordinator, *types.MockRegistry, *types.MockPerformLogProvider) {
		mr := types.NewMockRegistry(t)
		mp := types.NewMockPerformLogProvider(t)
		return &reportCoordinator{
			logger:     l,
			registry:   mr,
			logs:       mp,
			idBlocks:   pkgutil.NewCache[idBlocker](time.Second),
			activeKeys: pkgutil.NewCache[bool](time.Minute),
			minConfs:   1,
			chStop:     make(chan struct{}),
		}, mr, mp
	}

	// set up the mocks and mock data
	key1Block1 := chain.UpkeepKey("1|1")
	key1Block2 := chain.UpkeepKey("2|1")
	key1Block3 := chain.UpkeepKey("3|1")
	key1Block4 := chain.UpkeepKey("4|1")
	id1 := types.UpkeepIdentifier("1")
	bk2 := chain.BlockKey("2")
	bk3 := chain.BlockKey("3")
	bk15 := chain.BlockKey("15")

	t.Run("FilterBeforeAccept", func(t *testing.T) {
		rc, mr, _ := setup(t, log.New(io.Discard, "nil", 0))
		filter := rc.Filter()

		// calling filter at this point should return true because the key has not
		// yet been added to the filter
		assert.Equal(t, true, filter(key1Block1), "should not filter out key 1 at block 1: key has not been accepted")

		assert.Equal(t, true, filter(key1Block2), "should not filter out key 1 at block 2: key has not been accepted")

		// is transmission confirmed should also return true because the key has
		// not been set in the filter
		// this would block any transmissions for this upkeep key (block & id)
		assert.Equal(t, true, rc.IsTransmissionConfirmed(key1Block1), "transmit will return confirmed: key has not been accepted")

		mr.AssertExpectations(t)
	})

	t.Run("Accept", func(t *testing.T) {
		rc, mr, _ := setup(t, log.New(io.Discard, "nil", 0))

		assert.NoError(t, rc.Accept(key1Block1), "no error expected from accepting the key")
		assert.ErrorIs(t, rc.Accept(key1Block1), ErrKeyAlreadyAccepted, "key should not be accepted again and should return an error")

		mr.AssertExpectations(t)
	})

	t.Run("FilterAfterAccept", func(t *testing.T) {
		rc, mr, _ := setup(t, log.New(io.Discard, "nil", 0))
		filter := rc.Filter()

		_ = rc.Accept(key1Block1)

		// no logs have been read at this point so the upkeep key should be filtered
		// out at all block numbers
		assert.Equal(t, false, filter(key1Block1), "filter should return false to indicate key should be filtered out at block 1")

		assert.Equal(t, false, filter(key1Block2), "filter should return false to indicate key should be filtered out at block 2")

		assert.Equal(t, false, rc.IsTransmissionConfirmed(key1Block1), "transmit should not be confirmed: key is now set, but no logs have been identified")

		// returning true for an unset key prevents any node from transmitting a key that was never accepted
		assert.Equal(t, true, rc.IsTransmissionConfirmed(key1Block2), "transmit should be confirmed: key was not set for block 2")

		mr.AssertExpectations(t)
	})

	t.Run("CollectLogsWithMinConfirmations_LessThan", func(t *testing.T) {
		rc, mr, mp := setup(t, log.New(io.Discard, "nil", 0))
		filter := rc.Filter()

		_ = rc.Accept(key1Block1)

		mp.Mock.On("PerformLogs", mock.Anything).Return([]types.PerformLog{
			{Key: key1Block1, TransmitBlock: bk2, Confirmations: 0},
		}, nil).Once()

		rc.checkLogs()

		// perform log didn't have the threshold number of confirmations
		// making the key still locked at all blocks
		assert.Equal(t, false, filter(key1Block1), "filter should return false to indicate key should be filtered out at block 1")

		assert.Equal(t, false, filter(key1Block2), "filter should return false to indicate key should be filtered out at block 2")

		assert.Equal(t, false, rc.IsTransmissionConfirmed(key1Block1), "transmit should not be confirmed: the key is now set, but no logs have been identified")
		assert.Equal(t, true, rc.IsTransmissionConfirmed(key1Block2), "transmit should be confirmed: key was not set for block 2")

		mp.AssertExpectations(t)
		mr.AssertExpectations(t)
	})

	t.Run("CollectLogsWithMinConfirmations_GreaterThan", func(t *testing.T) {
		rc, mr, mp := setup(t, log.New(io.Discard, "nil", 0))
		filter := rc.Filter()

		_ = rc.Accept(key1Block1)

		mp.Mock.On("PerformLogs", mock.Anything).Return([]types.PerformLog{
			{Key: key1Block1, TransmitBlock: bk2, Confirmations: 1},
		}, nil).Once()

		rc.checkLogs()

		// because the transmit block is block 2, the filter should continue
		// to filter out key up to block 2
		assert.Equal(t, false, filter(key1Block1), "filter should return false to indicate key should be filtered out at block 1")

		assert.Equal(t, false, filter(key1Block2), "filter should return false to indicate key should be filtered out at block 2")

		assert.Equal(t, true, filter(key1Block3), "filter should return true to indicate key should not be filtered out at block 3")

		assert.Equal(t, true, rc.IsTransmissionConfirmed(key1Block1), "transmit should be confirmed")
		assert.Equal(t, true, rc.IsTransmissionConfirmed(key1Block2), "transmit should be confirmed: key was not set for block 2")

		assert.ErrorIs(t, rc.Accept(key1Block1), ErrKeyAlreadyAccepted, "key should not be accepted after transmit confirmed and should return an error")

		mp.AssertExpectations(t)
		mr.AssertExpectations(t)
	})

	t.Run("SameID_DifferentBlocks", func(t *testing.T) {
		rc, mr, mp := setup(t, log.New(io.Discard, "nil", 0))
		filter := rc.Filter()

		// 1. key 1|1 is Accepted
		_ = rc.Accept(key1Block1)

		// 1a. key 1|1 filter returns false
		// 1c. key 2|1 filter returns false
		// 1d. key 4|1 filter returns false
		// reason: the node sees id 1 as 'in-flight' and blocks for all block numbers
		assertFilter(t, key1Block1, false, filter)
		assertFilter(t, key1Block2, false, filter)
		assertFilter(t, key1Block4, false, filter)

		// 1b. key 1|1 transmit confirmed returns false
		assert.Equal(t, false, rc.IsTransmissionConfirmed(key1Block1), "1|1 transmit should not be confirmed")

		// 2. key 2|1 is Accepted (if other nodes produce report)
		_ = rc.Accept(key1Block2)

		// 2a. key 1|1 filter returns false
		// 2c. key 2|1 filter returns false
		// 2e. key 4|1 filter returns false
		// reason: the node still sees id 1 as 'in-flight' and blocks for all block numbers
		assertFilter(t, key1Block1, false, filter)
		assertFilter(t, key1Block2, false, filter)
		assertFilter(t, key1Block4, false, filter)

		// 2b. key 1|1 transmit confirmed returns false
		// 2d. key 2|1 transmit confirmed returns false
		assert.Equal(t, false, rc.IsTransmissionConfirmed(key1Block1), "1|1 transmit should not be confirmed")
		assert.Equal(t, false, rc.IsTransmissionConfirmed(key1Block2), "2|1 transmit should not be confirmed")

		// 3. perform log for 1|1 is at block 2
		mp.Mock.On("PerformLogs", mock.Anything).Return([]types.PerformLog{
			{Key: key1Block1, TransmitBlock: bk2, Confirmations: 1},
		}, nil).Once()

		rc.checkLogs()

		// 3a. key 1|1 filter returns false
		// 3c. key 2|1 filter returns false
		// 3e. key 4|1 filter returns false
		// reason: the node still sees id 1 as 'in-flight' and blocks for all block numbers
		assertFilter(t, key1Block1, false, filter)
		assertFilter(t, key1Block2, false, filter)
		assertFilter(t, key1Block4, false, filter)

		// 3b. key 1|1 transmit confirmed returns true
		// 3d. key 2|1 transmit confirmed returns false
		// reason: transmission for key 1|1 was in the logs, but not 2|1
		assert.Equal(t, true, rc.IsTransmissionConfirmed(key1Block1), "1|1 transmit should not be confirmed")
		assert.Equal(t, false, rc.IsTransmissionConfirmed(key1Block2), "2|1 transmit should not be confirmed")

		// 4. perform log for 2|1 is at block 3
		mp.Mock.On("PerformLogs", mock.Anything).Return([]types.PerformLog{
			{Key: key1Block2, TransmitBlock: bk3, Confirmations: 1},
		}, nil).Once()

		rc.checkLogs()

		// 4a. key 1|1 filter returns false
		// 4c. key 2|1 filter returns false
		// 4e. key 4|1 filter returns true
		// reason: the id unblocks after the highest block number seen
		assertFilter(t, key1Block1, false, filter)
		assertFilter(t, key1Block2, false, filter)
		assertFilter(t, key1Block4, true, filter)

		// 4b. key 1|1 transmit confirmed returns true
		// 4d. key 2|1 transmit confirmed returns true
		// reason: all transmissions have come in the logs
		assert.Equal(t, true, rc.IsTransmissionConfirmed(key1Block1), "1|1 transmit should not be confirmed")
		assert.Equal(t, true, rc.IsTransmissionConfirmed(key1Block2), "2|1 transmit should not be confirmed")

		mp.AssertExpectations(t)
		mr.AssertExpectations(t)
	})

	t.Run("Filter", func(t *testing.T) {
		rc, mr, _ := setup(t, log.New(io.Discard, "nil", 0))
		filter := rc.Filter()

		rc.idBlocks.Set(string(id1), idBlocker{
			TransmitBlockNumber: bk15,
		}, pkgutil.DefaultCacheExpiration)

		assert.False(t, filter(key1Block4))

		mr.AssertExpectations(t)
	})
}

func assertFilter(t *testing.T, key types.UpkeepKey, exp bool, f func(types.UpkeepKey) bool) {
	assert.Equal(t, exp, f(key), "filter should return '%v' to indicate key should not be filtered out at block %s", exp, key)
}
