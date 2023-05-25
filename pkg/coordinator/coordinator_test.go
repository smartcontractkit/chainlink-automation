package coordinator

import (
	"context"
	"fmt"
	"io"
	"log"
	"testing"
	"time"

	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg"
	"github.com/smartcontractkit/ocr2keepers/pkg/coordinator/mocks"
	"github.com/smartcontractkit/ocr2keepers/pkg/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestReportCoordinator(t *testing.T) {
	setup := func(t *testing.T, l *log.Logger) (*reportCoordinator, *mocks.Encoder, *mocks.LogProvider) {
		mEnc := new(mocks.Encoder)
		mLogs := new(mocks.LogProvider)

		return &reportCoordinator{
			logger:     l,
			logs:       mLogs,
			encoder:    mEnc,
			idBlocks:   util.NewCache[idBlocker](time.Second),
			activeKeys: util.NewCache[bool](time.Minute),
			minConfs:   1,
			chStop:     make(chan struct{}),
		}, mEnc, mLogs
	}

	// set up the mocks and mock data
	id1 := ocr2keepers.UpkeepIdentifier("1")

	key1Block1 := ocr2keepers.UpkeepKey("1|1")
	key1Block2 := ocr2keepers.UpkeepKey("2|1")
	key1Block3 := ocr2keepers.UpkeepKey("3|1")
	key1Block4 := ocr2keepers.UpkeepKey("4|1")

	bk1 := ocr2keepers.BlockKey("1")
	bk2 := ocr2keepers.BlockKey("2")
	bk3 := ocr2keepers.BlockKey("3")
	bk4 := ocr2keepers.BlockKey("4")
	bk15 := ocr2keepers.BlockKey("15")

	// seeds a test with a starting accepted block and asserts that
	// values are as expected
	seed := func(t *testing.T, rc *reportCoordinator, mr *mocks.Encoder) {
		// key 1|1 is Accepted
		mr.On("SplitUpkeepKey", mock.Anything).Return(bk1, id1, nil).Once()
		assert.NoError(t, rc.Accept(key1Block1))

		// the node sees id 1 as 'in-flight' and blocks for all block numbers
		mr.On("SplitUpkeepKey", key1Block1).Return(bk1, id1, nil).Once()
		mr.On("After", bk1, IndefiniteBlockingKey).Return(false, nil).Once()
		assertFilter(t, key1Block1, true, rc.IsPending)

		mr.On("SplitUpkeepKey", key1Block2).Return(bk2, id1, nil).Once()
		mr.On("After", bk2, IndefiniteBlockingKey).Return(false, nil).Once()
		assertFilter(t, key1Block2, true, rc.IsPending)

		mr.On("SplitUpkeepKey", key1Block4).Return(bk4, id1, nil).Once()
		mr.On("After", bk4, IndefiniteBlockingKey).Return(false, nil).Once()
		assertFilter(t, key1Block4, true, rc.IsPending)

		// key 1|1 transmit confirmed returns false
		assert.Equal(t, false, rc.IsTransmissionConfirmed(key1Block1), "1|1 transmit should not be confirmed")
	}

	t.Run("FilterBeforeAccept", func(t *testing.T) {
		rc, mr, _ := setup(t, log.New(io.Discard, "nil", 0))

		// calling filter at this point should return true because the key has not
		// yet been added to the filter
		mr.On("SplitUpkeepKey", mock.Anything).Return(ocr2keepers.BlockKey("1"), ocr2keepers.UpkeepIdentifier("1"), nil).Once()
		key1Block1Pending, err := rc.IsPending(key1Block1)
		assert.NoError(t, err)
		assert.Equal(t, false, key1Block1Pending, "should not filter out key 1 at block 1: key has not been accepted")

		mr.On("SplitUpkeepKey", mock.Anything).Return(ocr2keepers.BlockKey("2"), ocr2keepers.UpkeepIdentifier("1"), nil).Once()
		key1Block2Pending, err := rc.IsPending(key1Block2)
		assert.NoError(t, err)
		assert.Equal(t, false, key1Block2Pending, "should not filter out key 1 at block 2: key has not been accepted")

		// is transmission confirmed should also return true because the key has
		// not been set in the filter
		// this would block any transmissions for this upkeep key (block & id)
		assert.Equal(t, true, rc.IsTransmissionConfirmed(key1Block1), "transmit will return confirmed: key has not been accepted")

		mr.AssertExpectations(t)
	})

	t.Run("Accept", func(t *testing.T) {
		rc, mr, _ := setup(t, log.New(io.Discard, "nil", 0))

		mr.On("SplitUpkeepKey", mock.Anything).Return(ocr2keepers.BlockKey("1"), ocr2keepers.UpkeepIdentifier("1"), nil).Twice()
		assert.NoError(t, rc.Accept(key1Block1), "no error expected from accepting the key")
		assert.NoError(t, rc.Accept(key1Block1), "Key can get accepted again")

		mr.AssertExpectations(t)
	})

	t.Run("Accept errors on an error parsing BlockKeyAndUpkeepID", func(t *testing.T) {
		rc, mr, _ := setup(t, log.New(io.Discard, "nil", 0))

		key := ocr2keepers.UpkeepKey("||")
		err := fmt.Errorf("split error")

		mr.On("SplitUpkeepKey", key).Return(ocr2keepers.BlockKey(""), ocr2keepers.UpkeepIdentifier(""), err).Once()

		assert.ErrorIs(t, rc.Accept(key), err)

		mr.AssertExpectations(t)
	})

	t.Run("FilterAfterAccept", func(t *testing.T) {
		rc, mr, _ := setup(t, log.New(io.Discard, "nil", 0))

		mr.On("SplitUpkeepKey", mock.Anything).Return(ocr2keepers.BlockKey("1"), ocr2keepers.UpkeepIdentifier("1"), nil).Once()
		assert.NoError(t, rc.Accept(key1Block1))

		// no logs have been read at this point so the upkeep key should be filtered
		// out at all block numbers
		mr.On("SplitUpkeepKey", mock.Anything).Return(ocr2keepers.BlockKey("1"), ocr2keepers.UpkeepIdentifier("1"), nil).Once()
		mr.On("After", mock.Anything, IndefiniteBlockingKey).Return(false, nil).Once()
		key1Block1Pending, err := rc.IsPending(key1Block1)
		assert.NoError(t, err)
		assert.Equal(t, true, key1Block1Pending, "filter should return true to indicate key should be filtered out at block 1")

		mr.On("SplitUpkeepKey", mock.Anything).Return(ocr2keepers.BlockKey("2"), ocr2keepers.UpkeepIdentifier("1"), nil).Once()
		mr.On("After", mock.Anything, IndefiniteBlockingKey).Return(false, nil).Once()
		key1Block2Pending, err := rc.IsPending(key1Block2)
		assert.NoError(t, err)
		assert.Equal(t, true, key1Block2Pending, "filter should return true to indicate key should be filtered out at block 2")

		assert.Equal(t, false, rc.IsTransmissionConfirmed(key1Block1), "transmit should not be confirmed: key is now set, but no logs have been identified")
		// returning true for an unset key prevents any node from transmitting a key that was never accepted
		assert.Equal(t, true, rc.IsTransmissionConfirmed(key1Block2), "transmit should be confirmed: key was not set for block 2")

		mr.AssertExpectations(t)
	})

	t.Run("CollectLogsWithMinConfirmations_LessThan", func(t *testing.T) {
		rc, mr, mp := setup(t, log.New(io.Discard, "nil", 0))

		mr.On("SplitUpkeepKey", mock.Anything).Return(ocr2keepers.BlockKey("1"), ocr2keepers.UpkeepIdentifier("1"), nil).Once()
		assert.NoError(t, rc.Accept(key1Block1))

		mp.Mock.On("PerformLogs", mock.Anything).Return([]ocr2keepers.PerformLog{
			{Key: key1Block1, TransmitBlock: bk2, Confirmations: 0}, // not enough confirmations; log won't apply
		}, nil).Once()
		mp.Mock.On("StaleReportLogs", mock.Anything).Return([]ocr2keepers.StaleReportLog{}, nil).Once()

		assert.NoError(t, rc.checkLogs(context.Background()))

		// perform log didn't have the threshold number of confirmations
		// making the key still locked at all blocks
		mr.On("SplitUpkeepKey", mock.Anything).Return(ocr2keepers.BlockKey("1"), ocr2keepers.UpkeepIdentifier("1"), nil).Once()
		mr.On("After", mock.Anything, IndefiniteBlockingKey).Return(false, nil).Once()
		key1Block1Pending, err := rc.IsPending(key1Block1)
		assert.NoError(t, err)
		assert.Equal(t, true, key1Block1Pending, "filter should return true to indicate key should be filtered out at block 1")

		mr.On("SplitUpkeepKey", mock.Anything).Return(ocr2keepers.BlockKey("2"), ocr2keepers.UpkeepIdentifier("1"), nil).Once()
		mr.On("After", mock.Anything, IndefiniteBlockingKey).Return(false, nil).Once()
		key1Block2Pending, err := rc.IsPending(key1Block2)
		assert.NoError(t, err)
		assert.Equal(t, true, key1Block2Pending, "filter should return true to indicate key should be filtered out at block 2")

		assert.Equal(t, false, rc.IsTransmissionConfirmed(key1Block1), "transmit should not be confirmed: the key is now set, but no logs have been identified")
		assert.Equal(t, true, rc.IsTransmissionConfirmed(key1Block2), "transmit should be confirmed: key was not set for block 2")

		mp.AssertExpectations(t)
		mr.AssertExpectations(t)
	})

	t.Run("CollectLogsWithMinConfirmations_GreaterThan", func(t *testing.T) {
		rc, mr, mp := setup(t, log.New(io.Discard, "nil", 0))

		mr.On("SplitUpkeepKey", mock.Anything).Return(bk1, ocr2keepers.UpkeepIdentifier("1"), nil).Once()
		assert.NoError(t, rc.Accept(key1Block1))

		mp.Mock.On("PerformLogs", mock.Anything).Return([]ocr2keepers.PerformLog{
			{Key: key1Block1, TransmitBlock: bk2, Confirmations: 1}, // log has minimum confirmations and will apply
		}, nil).Once()
		mp.Mock.On("StaleReportLogs", mock.Anything).Return([]ocr2keepers.StaleReportLog{}, nil).Once()

		mr.On("SplitUpkeepKey", mock.Anything).Return(bk1, ocr2keepers.UpkeepIdentifier("1"), nil).Once()
		mr.On("After", bk1, bk1).Return(false, nil).Once()
		mr.On("After", bk1, bk1).Return(false, nil).Once()

		assert.NoError(t, rc.checkLogs(context.Background()))

		// because the transmit block is block 2, the filter should continue
		// to filter out key up to block 2
		mr.On("SplitUpkeepKey", mock.Anything).Return(bk1, ocr2keepers.UpkeepIdentifier("1"), nil).Once()
		mr.On("After", bk1, bk2).Return(false, nil).Once()
		key1Block1Pending, err := rc.IsPending(key1Block1)
		assert.NoError(t, err)
		assert.Equal(t, true, key1Block1Pending, "filter should return true to indicate key should be filtered out at block 1")

		mr.On("SplitUpkeepKey", mock.Anything).Return(bk2, ocr2keepers.UpkeepIdentifier("1"), nil).Once()
		mr.On("After", bk2, bk2).Return(false, nil).Once()
		key1Block2Pending, err := rc.IsPending(key1Block2)
		assert.NoError(t, err)
		assert.Equal(t, true, key1Block2Pending, "filter should return true to indicate key should be filtered out at block 2")

		mr.On("SplitUpkeepKey", mock.Anything).Return(bk3, ocr2keepers.UpkeepIdentifier("1"), nil).Once()
		mr.On("After", bk3, bk2).Return(true, nil).Once()
		key1Block3Pending, err := rc.IsPending(key1Block3)
		assert.NoError(t, err)
		assert.Equal(t, false, key1Block3Pending, "filter should return false to indicate key should not be filtered out at block 3")

		assert.Equal(t, true, rc.IsTransmissionConfirmed(key1Block1), "transmit should be confirmed")
		assert.Equal(t, true, rc.IsTransmissionConfirmed(key1Block2), "transmit should be confirmed: key was not set for block 2")

		// Accpeting the key again should not affect the filters
		mr.On("SplitUpkeepKey", mock.Anything).Return(bk1, ocr2keepers.UpkeepIdentifier("1"), nil).Once()
		assert.NoError(t, rc.Accept(key1Block1), "Key can get accepted again")

		mr.On("SplitUpkeepKey", mock.Anything).Return(bk2, ocr2keepers.UpkeepIdentifier("1"), nil).Once()
		mr.On("After", bk2, bk2).Return(false, nil).Once()
		key1Block2Pending, err = rc.IsPending(key1Block2)
		assert.NoError(t, err)
		assert.Equal(t, true, key1Block2Pending, "filter should return true to indicate key should be filtered out at block 2")

		mr.On("SplitUpkeepKey", mock.Anything).Return(bk3, ocr2keepers.UpkeepIdentifier("1"), nil).Once()
		mr.On("After", bk3, bk2).Return(true, nil).Once()
		key1Block3Pending, err = rc.IsPending(key1Block3)
		assert.NoError(t, err)
		assert.Equal(t, false, key1Block3Pending, "filter should return false to indicate key should not be filtered out at block 3")

		mp.AssertExpectations(t)
		mr.AssertExpectations(t)
	})

	t.Run("SameID_DifferentBlocks", func(t *testing.T) {
		rc, mr, mp := setup(t, log.New(io.Discard, "nil", 0))
		filter := rc.IsPending

		seed(t, rc, mr)

		// 2. key 2|1 is Accepted (if other nodes produce report)
		mr.On("SplitUpkeepKey", mock.Anything).Return(bk2, id1, nil).Once()
		mr.On("After", bk2, bk1).Return(true, nil).Once()
		assert.NoError(t, rc.Accept(key1Block2))

		// 2a. key 1|1 IsPending returns true
		// 2c. key 2|1 IsPending returns true
		// 2e. key 4|1 IsPending returns true
		// reason: the node still sees id 1 as 'in-flight' and blocks for all block numbers
		mr.On("SplitUpkeepKey", key1Block1).Return(bk1, id1, nil).Once()
		mr.On("After", bk1, IndefiniteBlockingKey).Return(false, nil).Once()
		assertFilter(t, key1Block1, true, filter)

		mr.On("SplitUpkeepKey", key1Block2).Return(bk2, id1, nil).Once()
		mr.On("After", bk2, IndefiniteBlockingKey).Return(false, nil).Once()
		assertFilter(t, key1Block2, true, filter)

		mr.On("SplitUpkeepKey", key1Block4).Return(bk4, id1, nil).Once()
		mr.On("After", bk4, IndefiniteBlockingKey).Return(false, nil).Once()
		assertFilter(t, key1Block4, true, filter)

		// 2b. key 1|1 transmit confirmed returns false
		// 2d. key 2|1 transmit confirmed returns false
		assert.Equal(t, false, rc.IsTransmissionConfirmed(key1Block1), "1|1 transmit should not be confirmed")
		assert.Equal(t, false, rc.IsTransmissionConfirmed(key1Block2), "2|1 transmit should not be confirmed")

		// 3. perform log for 1|1 is at block 2
		mp.Mock.On("PerformLogs", mock.Anything).Return([]ocr2keepers.PerformLog{
			{Key: key1Block1, TransmitBlock: bk2, Confirmations: 1},
		}, nil).Once()
		mp.Mock.On("StaleReportLogs", mock.Anything).Return([]ocr2keepers.StaleReportLog{}, nil).Once()

		mr.On("SplitUpkeepKey", mock.Anything).Return(bk1, id1, nil).Once()
		mr.On("After", bk1, bk2).Return(false, nil).Once()
		mr.On("After", bk2, bk1).Return(true, nil).Once()

		assert.NoError(t, rc.checkLogs(context.Background()))

		// 3a. key 1|1 IsPending returns true
		// 3c. key 2|1 IsPending returns true
		// 3e. key 4|1 IsPending returns true
		// reason: the node still sees id 1 as 'in-flight' and blocks for all block numbers
		mr.On("SplitUpkeepKey", key1Block1).Return(bk1, id1, nil).Once()
		mr.On("After", bk1, IndefiniteBlockingKey).Return(false, nil).Once()
		assertFilter(t, key1Block1, true, filter)

		mr.On("SplitUpkeepKey", key1Block2).Return(bk2, id1, nil).Once()
		mr.On("After", bk2, IndefiniteBlockingKey).Return(false, nil).Once()
		assertFilter(t, key1Block2, true, filter)

		mr.On("SplitUpkeepKey", key1Block4).Return(bk4, id1, nil).Once()
		mr.On("After", bk4, IndefiniteBlockingKey).Return(false, nil).Once()
		assertFilter(t, key1Block4, true, filter)

		// 3b. key 1|1 transmit confirmed returns true
		// 3d. key 2|1 transmit confirmed returns false
		// reason: transmission for key 1|1 was in the logs, but not 2|1
		assert.Equal(t, true, rc.IsTransmissionConfirmed(key1Block1), "1|1 transmit should not be confirmed")
		assert.Equal(t, false, rc.IsTransmissionConfirmed(key1Block2), "2|1 transmit should not be confirmed")

		// 4. perform log for 2|1 is at block 3
		mp.Mock.On("PerformLogs", mock.Anything).Return([]ocr2keepers.PerformLog{
			{Key: key1Block2, TransmitBlock: bk3, Confirmations: 1},
		}, nil).Once()
		mp.Mock.On("StaleReportLogs", mock.Anything).Return([]ocr2keepers.StaleReportLog{}, nil).Once()

		mr.On("SplitUpkeepKey", mock.Anything).Return(bk2, id1, nil).Once()
		mr.On("After", bk2, bk2).Return(false, nil).Once()
		mr.On("After", bk2, bk2).Return(false, nil).Once()

		assert.NoError(t, rc.checkLogs(context.Background()))

		// 4a. key 1|1 IsPending returns true
		// 4c. key 2|1 IsPending returns true
		// 4e. key 4|1 IsPending returns false
		// reason: the id unblocks after the highest block number seen
		mr.On("SplitUpkeepKey", key1Block1).Return(bk1, id1, nil).Once()
		mr.On("After", bk1, bk3).Return(false, nil).Once()
		assertFilter(t, key1Block1, true, filter)

		mr.On("SplitUpkeepKey", key1Block2).Return(bk2, id1, nil).Once()
		mr.On("After", bk2, bk3).Return(false, nil).Once()
		assertFilter(t, key1Block2, true, filter)

		mr.On("SplitUpkeepKey", key1Block4).Return(bk4, id1, nil).Once()
		mr.On("After", bk4, bk3).Return(true, nil).Once()
		assertFilter(t, key1Block4, false, filter)

		// 4b. key 1|1 transmit confirmed returns true
		// 4d. key 2|1 transmit confirmed returns true
		// reason: all transmissions have come in the logs
		assert.Equal(t, true, rc.IsTransmissionConfirmed(key1Block1), "1|1 transmit should not be confirmed")
		assert.Equal(t, true, rc.IsTransmissionConfirmed(key1Block2), "2|1 transmit should not be confirmed")

		mp.AssertExpectations(t)
		mr.AssertExpectations(t)
	})

	t.Run("Reorged Perform Logs", func(t *testing.T) {
		rc, mr, mp := setup(t, log.New(io.Discard, "nil", 0))
		filter := rc.IsPending

		seed(t, rc, mr)

		// perform log for 1|1 is at block 2
		mp.Mock.On("PerformLogs", mock.Anything).Return([]ocr2keepers.PerformLog{
			{Key: key1Block1, TransmitBlock: bk2, Confirmations: 1},
		}, nil).Once()
		mp.Mock.On("StaleReportLogs", mock.Anything).Return([]ocr2keepers.StaleReportLog{}, nil).Once()

		mr.On("SplitUpkeepKey", mock.Anything).Return(bk1, id1, nil).Once()
		mr.On("After", bk1, bk1).Return(false, nil).Once()
		mr.On("After", bk1, bk1).Return(false, nil).Once()

		assert.NoError(t, rc.checkLogs(context.Background()))

		// Transmit should be confirmed as perform log is found
		assert.Equal(t, true, rc.IsTransmissionConfirmed(key1Block1), "1|1 transmit should be confirmed")

		// key 1|1 IsPending returns true
		// key 2|1 IsPending returns true
		// key 3|1 IsPending returns false
		// reason: the node unblocks id 1 after block 2
		mr.On("SplitUpkeepKey", key1Block1).Return(bk1, id1, nil).Once()
		mr.On("After", bk1, bk2).Return(false, nil).Once()
		assertFilter(t, key1Block1, true, filter)

		mr.On("SplitUpkeepKey", key1Block2).Return(bk2, id1, nil).Once()
		mr.On("After", bk2, bk2).Return(false, nil).Once()
		assertFilter(t, key1Block2, true, filter)

		mr.On("SplitUpkeepKey", key1Block3).Return(bk3, id1, nil).Once()
		mr.On("After", bk3, bk2).Return(true, nil).Once()
		assertFilter(t, key1Block3, false, filter)

		// A re-orged perform log for 1|1 is found at block 3
		mp.Mock.On("PerformLogs", mock.Anything).Return([]ocr2keepers.PerformLog{
			{Key: key1Block1, TransmitBlock: bk3, Confirmations: 1},
		}, nil).Once()
		mp.Mock.On("StaleReportLogs", mock.Anything).Return([]ocr2keepers.StaleReportLog{}, nil).Once()

		mr.On("SplitUpkeepKey", mock.Anything).Return(bk1, id1, nil).Once()
		mr.On("After", bk1, bk1).Return(false, nil).Once()
		mr.On("After", bk1, bk1).Return(false, nil).Once()
		mr.On("After", bk3, bk2).Return(true, nil).Once()

		assert.NoError(t, rc.checkLogs(context.Background()))

		// Transmit confirmed should not change
		assert.Equal(t, true, rc.IsTransmissionConfirmed(key1Block1), "1|1 transmit should be confirmed")

		// key 1|1 IsPending returns true
		// key 2|1 IsPending returns true
		// key 3|1 IsPending returns true
		// key 4|1 IsPending returns false
		// reason: the node unblocks id 1 after block 3 (latest reorged perform)
		mr.On("SplitUpkeepKey", key1Block1).Return(bk1, id1, nil).Once()
		mr.On("After", bk1, bk3).Return(false, nil).Once()
		assertFilter(t, key1Block1, true, filter)

		mr.On("SplitUpkeepKey", key1Block2).Return(bk2, id1, nil).Once()
		mr.On("After", bk2, bk3).Return(false, nil).Once()
		assertFilter(t, key1Block2, true, filter)

		mr.On("SplitUpkeepKey", key1Block3).Return(bk3, id1, nil).Once()
		mr.On("After", bk3, bk3).Return(false, nil).Once()
		assertFilter(t, key1Block3, true, filter)

		mr.On("SplitUpkeepKey", key1Block4).Return(bk4, id1, nil).Once()
		mr.On("After", bk4, bk3).Return(true, nil).Once()
		assertFilter(t, key1Block4, false, filter)

		mp.AssertExpectations(t)
		mr.AssertExpectations(t)
	})

	t.Run("Same key accepted twice", func(t *testing.T) {
		rc, mr, mp := setup(t, log.New(io.Discard, "nil", 0))
		filter := rc.IsPending

		seed(t, rc, mr)

		// key 1|1 is Accepted again. It should not error out
		mr.On("SplitUpkeepKey", mock.Anything).Return(bk1, id1, nil).Once()
		assert.NoError(t, rc.Accept(key1Block1))

		// Same filtering and transmission confirmed should hold true
		mr.On("SplitUpkeepKey", key1Block1).Return(bk1, id1, nil).Once()
		mr.On("After", bk1, IndefiniteBlockingKey).Return(false, nil).Once()
		assertFilter(t, key1Block1, true, filter)

		mr.On("SplitUpkeepKey", key1Block2).Return(bk2, id1, nil).Once()
		mr.On("After", bk2, IndefiniteBlockingKey).Return(false, nil).Once()
		assertFilter(t, key1Block2, true, filter)

		mr.On("SplitUpkeepKey", key1Block4).Return(bk4, id1, nil).Once()
		mr.On("After", bk4, IndefiniteBlockingKey).Return(false, nil).Once()
		assertFilter(t, key1Block4, true, filter)

		assert.Equal(t, false, rc.IsTransmissionConfirmed(key1Block1), "1|1 transmit should not be confirmed")

		// perform log for 1|1 is found at block 2
		mp.Mock.On("PerformLogs", mock.Anything).Return([]ocr2keepers.PerformLog{
			{Key: key1Block1, TransmitBlock: bk2, Confirmations: 1},
		}, nil).Once()
		mp.Mock.On("StaleReportLogs", mock.Anything).Return([]ocr2keepers.StaleReportLog{}, nil).Once()

		mr.On("SplitUpkeepKey", mock.Anything).Return(bk1, id1, nil).Once()
		mr.On("After", bk1, bk1).Return(false, nil).Once()
		mr.On("After", bk1, bk1).Return(false, nil).Once()

		assert.NoError(t, rc.checkLogs(context.Background()))

		// reason: the node unblocks id 1 after block 2
		mr.On("SplitUpkeepKey", key1Block1).Return(bk1, id1, nil).Once()
		mr.On("After", bk1, bk2).Return(false, nil).Once()
		assertFilter(t, key1Block1, true, filter)

		mr.On("SplitUpkeepKey", key1Block2).Return(bk2, id1, nil).Once()
		mr.On("After", bk2, bk2).Return(false, nil).Once()
		assertFilter(t, key1Block2, true, filter)

		mr.On("SplitUpkeepKey", key1Block3).Return(bk3, id1, nil).Once()
		mr.On("After", bk3, bk2).Return(true, nil).Once()
		assertFilter(t, key1Block3, false, filter)

		// Transmit should be confirmed as perform log is found
		assert.Equal(t, true, rc.IsTransmissionConfirmed(key1Block1), "1|1 transmit should be confirmed")

		// key 1|1 is Accepted again. It should not error out and not change filters
		mr.On("SplitUpkeepKey", mock.Anything).Return(bk1, id1, nil).Once()
		assert.NoError(t, rc.Accept(key1Block1))

		mr.On("SplitUpkeepKey", key1Block1).Return(bk1, id1, nil).Once()
		mr.On("After", bk1, bk2).Return(false, nil).Once()
		assertFilter(t, key1Block1, true, filter)

		mr.On("SplitUpkeepKey", key1Block2).Return(bk2, id1, nil).Once()
		mr.On("After", bk2, bk2).Return(false, nil).Once()
		assertFilter(t, key1Block2, true, filter)

		mr.On("SplitUpkeepKey", key1Block3).Return(bk3, id1, nil).Once()
		mr.On("After", bk3, bk2).Return(true, nil).Once()
		assertFilter(t, key1Block3, false, filter)

		assert.Equal(t, true, rc.IsTransmissionConfirmed(key1Block1), "1|1 transmit should be confirmed")

		// Now a new key is accepted which is after previously accepted key
		mr.On("SplitUpkeepKey", mock.Anything).Return(bk2, id1, nil).Once()
		mr.On("After", bk2, bk1).Return(true, nil).Once()
		assert.NoError(t, rc.Accept(key1Block2))

		assert.Equal(t, false, rc.IsTransmissionConfirmed(key1Block2), "2|1 transmit should not be confirmed")

		// Id should be blocked indefintely on all blocks
		mr.On("SplitUpkeepKey", key1Block3).Return(bk3, id1, nil).Once()
		mr.On("After", bk3, IndefiniteBlockingKey).Return(false, nil).Once()
		assertFilter(t, key1Block3, true, filter)

		mr.On("SplitUpkeepKey", key1Block4).Return(bk4, id1, nil).Once()
		mr.On("After", bk4, IndefiniteBlockingKey).Return(false, nil).Once()
		assertFilter(t, key1Block4, true, filter)

		mp.AssertExpectations(t)
		mr.AssertExpectations(t)
	})

	t.Run("Stale report log is found", func(t *testing.T) {
		rc, mr, mp := setup(t, log.New(io.Discard, "nil", 0))
		filter := rc.IsPending

		seed(t, rc, mr)

		// stale report log for 1|1 is found at block 4
		mp.Mock.On("PerformLogs", mock.Anything).Return([]ocr2keepers.PerformLog{}, nil).Once()
		mp.Mock.On("StaleReportLogs", mock.Anything).Return([]ocr2keepers.StaleReportLog{
			{Key: key1Block1, TransmitBlock: bk4, Confirmations: 1},
		}, nil).Once()

		mr.On("SplitUpkeepKey", mock.Anything).Return(bk1, id1, nil).Once()
		mr.On("Increment", bk1).Return(bk2, nil).Once()
		mr.On("After", bk1, bk1).Return(false, nil).Once()
		mr.On("After", bk1, bk1).Return(false, nil).Once()

		assert.NoError(t, rc.checkLogs(context.Background()))

		// reason: the node unblocks id 1 after block 2 (checkBlock(1) + 1)
		mr.On("SplitUpkeepKey", key1Block1).Return(bk1, id1, nil).Once()
		mr.On("After", bk1, bk2).Return(false, nil).Once()
		assertFilter(t, key1Block1, true, filter)

		mr.On("SplitUpkeepKey", key1Block2).Return(bk2, id1, nil).Once()
		mr.On("After", bk2, bk2).Return(false, nil).Once()
		assertFilter(t, key1Block2, true, filter)

		mr.On("SplitUpkeepKey", key1Block3).Return(bk3, id1, nil).Once()
		mr.On("After", bk3, bk2).Return(true, nil).Once()
		assertFilter(t, key1Block3, false, filter)

		mr.On("SplitUpkeepKey", key1Block4).Return(bk4, id1, nil).Once()
		mr.On("After", bk4, bk2).Return(true, nil).Once()
		assertFilter(t, key1Block4, false, filter)

		// Transmit should be confirmed as stale report log is found
		assert.Equal(t, true, rc.IsTransmissionConfirmed(key1Block1), "1|1 transmit should be confirmed")

		mp.AssertExpectations(t)
		mr.AssertExpectations(t)
	})

	t.Run("Perform log gets reorged to stale report log", func(t *testing.T) {
		rc, mr, mp := setup(t, log.New(io.Discard, "nil", 0))
		filter := rc.IsPending

		seed(t, rc, mr)

		// perform log for 1|1 is found at block 3
		mp.Mock.On("PerformLogs", mock.Anything).Return([]ocr2keepers.PerformLog{
			{Key: key1Block1, TransmitBlock: bk3, Confirmations: 1},
		}, nil).Once()
		mp.Mock.On("StaleReportLogs", mock.Anything).Return([]ocr2keepers.StaleReportLog{}, nil).Once()

		mr.On("SplitUpkeepKey", mock.Anything).Return(bk1, id1, nil).Once()
		mr.On("After", bk1, bk1).Return(false, nil).Once()
		mr.On("After", bk1, bk1).Return(false, nil).Once()

		assert.NoError(t, rc.checkLogs(context.Background()))

		// reason: the node unblocks id 1 after block 3
		mr.On("SplitUpkeepKey", key1Block1).Return(bk1, id1, nil).Once()
		mr.On("After", bk1, bk3).Return(false, nil).Once()
		assertFilter(t, key1Block1, true, filter)

		mr.On("SplitUpkeepKey", key1Block2).Return(bk2, id1, nil).Once()
		mr.On("After", bk2, bk3).Return(false, nil).Once()
		assertFilter(t, key1Block2, true, filter)

		mr.On("SplitUpkeepKey", key1Block3).Return(bk3, id1, nil).Once()
		mr.On("After", bk3, bk3).Return(false, nil).Once()
		assertFilter(t, key1Block3, true, filter)

		mr.On("SplitUpkeepKey", key1Block4).Return(bk4, id1, nil).Once()
		mr.On("After", bk4, bk3).Return(true, nil).Once()
		assertFilter(t, key1Block4, false, filter)

		// Transmit should be confirmed as perform log is found
		assert.Equal(t, true, rc.IsTransmissionConfirmed(key1Block1), "1|1 transmit should be confirmed")

		// Now the perform log gets re-orged into a stale report log on block 4
		// It should not cause amny changes in the filter as checkBlockNumber of stale report log
		// is still 1
		mp.Mock.On("PerformLogs", mock.Anything).Return([]ocr2keepers.PerformLog{}, nil).Once()
		mp.Mock.On("StaleReportLogs", mock.Anything).Return([]ocr2keepers.StaleReportLog{
			{Key: key1Block1, TransmitBlock: bk4, Confirmations: 1},
		}, nil).Once()

		mr.On("SplitUpkeepKey", mock.Anything).Return(bk1, id1, nil).Once()
		mr.On("Increment", bk1).Return(bk2, nil).Once()
		mr.On("After", bk2, bk3).Return(false, nil).Once()
		mr.On("After", bk1, bk1).Return(false, nil).Once()
		mr.On("After", bk1, bk1).Return(false, nil).Once()

		assert.NoError(t, rc.checkLogs(context.Background()))

		mr.On("SplitUpkeepKey", key1Block1).Return(bk1, id1, nil).Once()
		mr.On("After", bk1, bk3).Return(false, nil).Once()
		assertFilter(t, key1Block1, true, filter)

		mr.On("SplitUpkeepKey", key1Block2).Return(bk2, id1, nil).Once()
		mr.On("After", bk2, bk3).Return(false, nil).Once()
		assertFilter(t, key1Block2, true, filter)

		mr.On("SplitUpkeepKey", key1Block3).Return(bk3, id1, nil).Once()
		mr.On("After", bk3, bk3).Return(false, nil).Once()
		assertFilter(t, key1Block3, true, filter)

		mr.On("SplitUpkeepKey", key1Block4).Return(bk4, id1, nil).Once()
		mr.On("After", bk4, bk3).Return(true, nil).Once()
		assertFilter(t, key1Block4, false, filter)

		assert.Equal(t, true, rc.IsTransmissionConfirmed(key1Block1), "1|1 transmit should be confirmed")

		mp.AssertExpectations(t)
		mr.AssertExpectations(t)
	})

	t.Run("Stale report log gets reorged", func(t *testing.T) {
		rc, mr, mp := setup(t, log.New(io.Discard, "nil", 0))
		filter := rc.IsPending

		seed(t, rc, mr)

		// stale log for 1|1 is found at block 2
		mp.Mock.On("PerformLogs", mock.Anything).Return([]ocr2keepers.PerformLog{}, nil).Once()
		mp.Mock.On("StaleReportLogs", mock.Anything).Return([]ocr2keepers.StaleReportLog{
			{Key: key1Block1, TransmitBlock: bk2, Confirmations: 1},
		}, nil).Once()

		mr.On("SplitUpkeepKey", mock.Anything).Return(bk1, id1, nil).Once()
		mr.On("Increment", bk1).Return(bk2, nil).Once()
		mr.On("After", bk1, bk1).Return(false, nil).Once()
		mr.On("After", bk1, bk1).Return(false, nil).Once()

		assert.NoError(t, rc.checkLogs(context.Background()))

		// reason: the node unblocks id 1 after block 2
		mr.On("SplitUpkeepKey", key1Block1).Return(bk1, id1, nil).Once()
		mr.On("After", bk1, bk2).Return(false, nil).Once()
		assertFilter(t, key1Block1, true, filter)

		mr.On("SplitUpkeepKey", key1Block2).Return(bk2, id1, nil).Once()
		mr.On("After", bk2, bk2).Return(false, nil).Once()
		assertFilter(t, key1Block2, true, filter)

		mr.On("SplitUpkeepKey", key1Block3).Return(bk3, id1, nil).Once()
		mr.On("After", bk3, bk2).Return(true, nil).Once()
		assertFilter(t, key1Block3, false, filter)

		// Transmit should be confirmed as perform log is found
		assert.Equal(t, true, rc.IsTransmissionConfirmed(key1Block1), "1|1 transmit should be confirmed")

		// stale log for 1|1 is again found at block 4
		mp.Mock.On("PerformLogs", mock.Anything).Return([]ocr2keepers.PerformLog{}, nil).Once()
		mp.Mock.On("StaleReportLogs", mock.Anything).Return([]ocr2keepers.StaleReportLog{
			{Key: key1Block1, TransmitBlock: bk4, Confirmations: 1},
		}, nil).Once()

		mr.On("SplitUpkeepKey", mock.Anything).Return(bk1, id1, nil).Once()
		mr.On("Increment", bk1).Return(bk2, nil).Once()

		assert.NoError(t, rc.checkLogs(context.Background()))

		// Filters should not change as checkBlockNumber of stale report log remains unchanged
		mr.On("SplitUpkeepKey", key1Block1).Return(bk1, id1, nil).Once()
		mr.On("After", bk1, bk2).Return(false, nil).Once()
		assertFilter(t, key1Block1, true, filter)

		mr.On("SplitUpkeepKey", key1Block2).Return(bk2, id1, nil).Once()
		mr.On("After", bk2, bk2).Return(false, nil).Once()
		assertFilter(t, key1Block2, true, filter)

		mr.On("SplitUpkeepKey", key1Block3).Return(bk3, id1, nil).Once()
		mr.On("After", bk3, bk2).Return(true, nil).Once()
		assertFilter(t, key1Block3, false, filter)

		assert.Equal(t, true, rc.IsTransmissionConfirmed(key1Block1), "1|1 transmit should be confirmed")

		mp.AssertExpectations(t)
		mr.AssertExpectations(t)
	})

	t.Run("Multiple accepted keys and old ones get perform/stale report log", func(t *testing.T) {
		rc, mr, mp := setup(t, log.New(io.Discard, "nil", 0))
		filter := rc.IsPending

		seed(t, rc, mr)

		// Another key 2|1 is Accepted before receiving logs (This can happen if this node is lagging the network)
		mr.On("SplitUpkeepKey", mock.Anything).Return(bk2, id1, nil).Once()
		mr.On("After", bk2, bk1).Return(true, nil).Once()
		assert.NoError(t, rc.Accept(key1Block2))

		// the node sees id 1 as 'in-flight' and blocks for all block numbers
		mr.On("SplitUpkeepKey", key1Block1).Return(bk1, id1, nil).Once()
		mr.On("After", bk1, IndefiniteBlockingKey).Return(false, nil).Once()
		assertFilter(t, key1Block1, true, filter)

		mr.On("SplitUpkeepKey", key1Block2).Return(bk2, id1, nil).Once()
		mr.On("After", bk2, IndefiniteBlockingKey).Return(false, nil).Once()
		assertFilter(t, key1Block2, true, filter)

		mr.On("SplitUpkeepKey", key1Block4).Return(bk4, id1, nil).Once()
		mr.On("After", bk4, IndefiniteBlockingKey).Return(false, nil).Once()
		assertFilter(t, key1Block4, true, filter)

		// key 2|1 transmit confirmed returns false
		assert.Equal(t, false, rc.IsTransmissionConfirmed(key1Block2), "2|1 transmit should not be confirmed")

		// Now a perform log is fetched for the previous key. It should not have effect on id filters as
		// that is locked on higher checkBlockNumber
		mp.Mock.On("PerformLogs", mock.Anything).Return([]ocr2keepers.PerformLog{
			{Key: key1Block1, TransmitBlock: bk3, Confirmations: 1},
		}, nil).Once()
		mp.Mock.On("StaleReportLogs", mock.Anything).Return([]ocr2keepers.StaleReportLog{}, nil).Once()

		mr.On("SplitUpkeepKey", mock.Anything).Return(bk1, id1, nil).Once()
		mr.On("After", bk1, bk2).Return(false, nil).Once()
		mr.On("After", bk2, bk1).Return(false, nil).Once()

		assert.NoError(t, rc.checkLogs(context.Background()))

		// the node sees id 1 as 'in-flight' and blocks for all block numbers
		mr.On("SplitUpkeepKey", key1Block1).Return(bk1, id1, nil).Once()
		mr.On("After", bk1, bk3).Return(false, nil).Once()
		assertFilter(t, key1Block1, true, filter)

		mr.On("SplitUpkeepKey", key1Block2).Return(bk2, id1, nil).Once()
		mr.On("After", bk2, bk3).Return(false, nil).Once()
		assertFilter(t, key1Block2, true, filter)

		mr.On("SplitUpkeepKey", key1Block4).Return(bk4, id1, nil).Once()
		mr.On("After", bk4, bk3).Return(true, nil).Once()
		assertFilter(t, key1Block4, false, filter)

		// key 1|1 transmit confirmed now returns true
		assert.Equal(t, true, rc.IsTransmissionConfirmed(key1Block1), "1|1 transmit should be confirmed")
		// key 2|1 transmit confirmed returns false
		assert.Equal(t, false, rc.IsTransmissionConfirmed(key1Block2), "2|1 transmit should not be confirmed")

		//Now the node sees perform log for latest accepted key. It should unblock the key from id filters
		mp.Mock.On("PerformLogs", mock.Anything).Return([]ocr2keepers.PerformLog{
			{Key: key1Block2, TransmitBlock: bk3, Confirmations: 1},
		}, nil).Once()
		mp.Mock.On("StaleReportLogs", mock.Anything).Return([]ocr2keepers.StaleReportLog{}, nil).Once()

		mr.On("SplitUpkeepKey", mock.Anything).Return(bk1, id1, nil).Once()
		mr.On("After", bk1, bk1).Return(true, nil).Once()

		assert.NoError(t, rc.checkLogs(context.Background()))

		// ID unblocked from block 4
		mr.On("SplitUpkeepKey", key1Block1).Return(bk1, id1, nil).Once()
		mr.On("After", bk1, bk3).Return(false, nil).Once()
		assertFilter(t, key1Block1, true, filter)

		mr.On("SplitUpkeepKey", key1Block2).Return(bk2, id1, nil).Once()
		mr.On("After", bk2, bk3).Return(false, nil).Once()
		assertFilter(t, key1Block2, true, filter)

		mr.On("SplitUpkeepKey", key1Block4).Return(bk4, id1, nil).Once()
		mr.On("After", bk4, bk3).Return(true, nil).Once()
		assertFilter(t, key1Block4, false, filter)

		assert.Equal(t, true, rc.IsTransmissionConfirmed(key1Block2), "2|1 transmit should be confirmed")

		mp.AssertExpectations(t)
		mr.AssertExpectations(t)
	})

	t.Run("Multiple accepted keys and out of order logs", func(t *testing.T) {
		rc, mr, mp := setup(t, log.New(io.Discard, "nil", 0))
		filter := rc.IsPending

		seed(t, rc, mr)

		// Another key 2|1 is Accepted before receiving logs (This can happen if this node is lagging the network)
		mr.On("SplitUpkeepKey", mock.Anything).Return(bk2, id1, nil).Once()
		mr.On("After", bk2, bk1).Return(true, nil).Once()
		assert.NoError(t, rc.Accept(key1Block2))

		// the node sees id 1 as 'in-flight' and blocks for all block numbers
		mr.On("SplitUpkeepKey", key1Block1).Return(bk1, id1, nil).Once()
		mr.On("After", bk1, IndefiniteBlockingKey).Return(false, nil).Once()
		assertFilter(t, key1Block1, true, filter)

		mr.On("SplitUpkeepKey", key1Block2).Return(bk2, id1, nil).Once()
		mr.On("After", bk2, IndefiniteBlockingKey).Return(false, nil).Once()
		assertFilter(t, key1Block2, true, filter)

		mr.On("SplitUpkeepKey", key1Block4).Return(bk4, id1, nil).Once()
		mr.On("After", bk4, IndefiniteBlockingKey).Return(false, nil).Once()
		assertFilter(t, key1Block4, true, filter)

		// key 2|1 transmit confirmed returns false
		assert.Equal(t, false, rc.IsTransmissionConfirmed(key1Block2), "2|1 transmit should not be confirmed")

		// Now a perform log is received for the latest key. It should unblock the idFilters
		mp.Mock.On("PerformLogs", mock.Anything).Return([]ocr2keepers.PerformLog{
			{Key: key1Block2, TransmitBlock: bk3, Confirmations: 1},
		}, nil).Once()
		mp.Mock.On("StaleReportLogs", mock.Anything).Return([]ocr2keepers.StaleReportLog{}, nil).Once()

		mr.On("SplitUpkeepKey", mock.Anything).Return(bk2, id1, nil).Once()
		mr.On("After", bk2, bk2).Return(true, nil).Once()

		assert.NoError(t, rc.checkLogs(context.Background()))

		// ID unblocked from block 4
		mr.On("SplitUpkeepKey", mock.Anything).Return(bk1, id1, nil).Once()
		mr.On("After", bk1, bk3).Return(false, nil).Once()
		assertFilter(t, key1Block1, true, filter)

		mr.On("SplitUpkeepKey", mock.Anything).Return(bk2, id1, nil).Once()
		mr.On("After", bk2, bk3).Return(false, nil).Once()
		assertFilter(t, key1Block2, true, filter)

		mr.On("SplitUpkeepKey", mock.Anything).Return(bk4, id1, nil).Once()
		mr.On("After", bk4, bk3).Return(true, nil).Once()
		assertFilter(t, key1Block4, false, filter)

		assert.Equal(t, false, rc.IsTransmissionConfirmed(key1Block1), "1|1 transmit should not be confirmed")
		assert.Equal(t, true, rc.IsTransmissionConfirmed(key1Block2), "2|1 transmit should be confirmed")

		//Now the node sees perform log for previous accepted key (out of order). It should not have any effect
		//on id filters
		mp.Mock.On("PerformLogs", mock.Anything).Return([]ocr2keepers.PerformLog{
			{Key: key1Block1, TransmitBlock: bk4, Confirmations: 1},
		}, nil).Once()
		mp.Mock.On("StaleReportLogs", mock.Anything).Return([]ocr2keepers.StaleReportLog{}, nil).Once()

		mr.On("SplitUpkeepKey", mock.Anything).Return(bk1, id1, nil).Once()
		mr.On("After", bk1, bk2).Return(false, nil).Once()
		mr.On("After", bk2, bk1).Return(false, nil).Once()
		mr.On("After", bk4, bk3).Return(true, nil).Once()

		assert.NoError(t, rc.checkLogs(context.Background()))

		mr.On("SplitUpkeepKey", mock.Anything).Return(bk1, id1, nil).Once()
		mr.On("After", bk1, bk4).Return(false, nil).Once()
		assertFilter(t, key1Block1, true, filter)

		mr.On("SplitUpkeepKey", mock.Anything).Return(bk2, id1, nil).Once()
		mr.On("After", bk2, bk4).Return(false, nil).Once()
		assertFilter(t, key1Block2, true, filter)

		mr.On("SplitUpkeepKey", mock.Anything).Return(bk4, id1, nil).Once()
		mr.On("After", bk4, bk4).Return(false, nil).Once()
		assertFilter(t, key1Block4, true, filter)

		assert.Equal(t, true, rc.IsTransmissionConfirmed(key1Block1), "1|1 transmit should be confirmed")
		assert.Equal(t, true, rc.IsTransmissionConfirmed(key1Block2), "2|1 transmit should be confirmed")

		mp.AssertExpectations(t)
		mr.AssertExpectations(t)
	})

	t.Run("Filter", func(t *testing.T) {
		t.Run("Determines that a key should be filtered out", func(t *testing.T) {
			rc, mr, _ := setup(t, log.New(io.Discard, "nil", 0))

			rc.idBlocks.Set(string(id1), idBlocker{
				TransmitBlockNumber: bk15,
			}, util.DefaultCacheExpiration)

			mr.On("SplitUpkeepKey", mock.Anything).Return(bk4, id1, nil).Once()
			mr.On("After", bk4, bk15).Return(false, nil).Once()
			pending, err := rc.IsPending(key1Block4)

			assert.NoError(t, err)
			assert.True(t, pending)

			mr.AssertExpectations(t)
		})

		t.Run("Determines that a key should be filtered out due to an error retrieving BlockKeyAndUpkeepID", func(t *testing.T) {
			rc, mr, _ := setup(t, log.New(io.Discard, "nil", 0))

			rc.idBlocks.Set(string(id1), idBlocker{
				TransmitBlockNumber: bk15,
			}, util.DefaultCacheExpiration)

			key := ocr2keepers.UpkeepKey("invalid")
			expected := fmt.Errorf("test")

			mr.On("SplitUpkeepKey", mock.Anything).Return(ocr2keepers.BlockKey(""), ocr2keepers.UpkeepIdentifier([]byte{}), expected).Once()
			pending, err := rc.IsPending(key)

			assert.ErrorIs(t, err, expected)
			assert.True(t, pending)

			mr.AssertExpectations(t)
		})

		t.Run("Determines that a key should be filtered out due to an error comparing block keys", func(t *testing.T) {
			rc, mr, _ := setup(t, log.New(io.Discard, "nil", 0))

			id := ocr2keepers.UpkeepIdentifier([]byte("1234"))
			key := ocr2keepers.UpkeepKey("1|1234")
			expected := fmt.Errorf("test")
			invalid := ocr2keepers.BlockKey("invalid")

			rc.idBlocks.Set("1234", idBlocker{
				TransmitBlockNumber: invalid,
			}, util.DefaultCacheExpiration)

			mr.On("SplitUpkeepKey", key).Return(bk1, id, nil).Once()
			mr.On("After", bk1, invalid).Return(false, expected).Once()
			pending, err := rc.IsPending(key)

			assert.ErrorIs(t, err, expected)
			assert.True(t, pending)

			mr.AssertExpectations(t)
		})
	})

}

func assertFilter(t *testing.T, key ocr2keepers.UpkeepKey, exp bool, f func(ocr2keepers.UpkeepKey) (bool, error)) {
	actual, err := f(key)
	assert.NoError(t, err)
	assert.Equal(t, exp, actual, "filter should return '%v' to indicate key should not be filtered out at block %s", exp, key)
}
