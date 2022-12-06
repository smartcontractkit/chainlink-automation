package keepers

import (
	"io"
	"log"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/smartcontractkit/ocr2keepers/internal/util"
	"github.com/smartcontractkit/ocr2keepers/pkg/types"
)

func TestNewFilter(t *testing.T) {
	mr := types.NewMockRegistry(t)
	mp := types.NewMockPerformLogProvider(t)

	l := log.New(io.Discard, "nil", 0)

	rc := &reportCoordinator{
		logger:     l,
		registry:   mr,
		logs:       mp,
		idBlocks:   util.NewCache[bool](time.Second),
		activeKeys: util.NewCache[bool](time.Minute),
		chStop:     make(chan struct{}),
	}

	// set up the mocks and mock data
	key := types.UpkeepKey("1|1")
	key2 := types.UpkeepKey("2|1")
	id := types.UpkeepIdentifier("1")

	// calling filter at this point should return true because the key has not
	// yet been added to the filter
	f := rc.Filter()
	mr.Mock.On("IdentifierFromKey", key).Return(id, nil)
	assert.Equal(t, true, f(key), "filter should not filter out key becase key has not been set")

	// is transmission confirmed should also return true because the key has
	// not been set in the filter
	assert.Equal(t, true, rc.IsTransmissionConfirmed(key), "transmit should be confirmed because key has not been set")

	mr.Mock.On("IdentifierFromKey", key).Return(id, nil)
	assert.NoError(t, rc.Accept(key), "no error expected from setting the key")
	assert.ErrorIs(t, rc.Accept(key), ErrKeyAlreadySet, "key should not be accepted again and should return an error")

	mr.Mock.On("IdentifierFromKey", key).Return(id, nil)
	assert.Equal(t, false, f(key), "key should be included in filter")
	assert.Equal(t, false, rc.IsTransmissionConfirmed(key), "transmit should not be confirmed because the key is now set, but no logs have been identified")

	mr.Mock.On("IdentifierFromKey", key).Return(id, nil)
	mr.Mock.On("IdentifierFromKey", key2).Return(id, nil)
	mp.Mock.On("PerformLogs", mock.Anything).Return([]types.PerformLog{
		{Key: key},
		{Key: key2},
	}, nil)

	// simulate starting the run process
	go rc.run()

	// wait for only 1 cycle to complete
	<-time.After(1100 * time.Millisecond)

	// stop the run process
	rc.chStop <- struct{}{}

	assert.Equal(t, true, rc.IsTransmissionConfirmed(key), "transmit should be confirmed after logs are read for the key")

	assert.ErrorIs(t, rc.Accept(key), ErrKeyAlreadySet, "key should not be accepted after transmit confirmed and should return an error")
	assert.NoError(t, rc.Accept(key2), "key2 should be able to accept since log should be ignored if it was not accepted before")

	mp.AssertExpectations(t)
	mr.AssertExpectations(t)
}
