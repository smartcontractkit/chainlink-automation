package keepers

import (
	"io"
	"log"
	"testing"
	"time"

	"github.com/smartcontractkit/ocr2keepers/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestNewFilter(t *testing.T) {
	mr := new(MockedRegistry)
	mp := new(MockedPerformLogProvider)

	l := log.New(io.Discard, "nil", 0)

	rc := newReportCoordinator(mr, time.Second, time.Minute, mp, l)

	// set up the mocks and mock data
	key := types.UpkeepKey([]byte("1|1"))
	id := types.UpkeepIdentifier([]byte("1"))
	mr.Mock.On("IdentifierFromKey", key).Return(id, nil).Times(6)

	// calling filter at this point should return true because the key has not
	// yet been added to the filter
	f := rc.Filter()
	assert.Equal(t, true, f(key), "filter should not filter out key becase key has not been set")

	// is transmission confirmed should also return true because the key has
	// not been set in the filter
	assert.Equal(t, true, rc.IsTransmissionConfirmed(key), "transmit should be confirmed because key has not been set")

	assert.NoError(t, rc.Accept(key), "no error expected from setting the key")
	assert.ErrorIs(t, rc.Accept(key), ErrKeyAlreadySet, "key should not be accepted again and should return an error")

	assert.Equal(t, false, f(key), "key should be included in filter")
	assert.Equal(t, false, rc.IsTransmissionConfirmed(key), "transmit should not be confirmed because the key is now set, but no logs have been identified")

	mp.Mock.On("PerformLogs", mock.Anything).Return([]types.PerformLog{
		{Key: key},
	}, nil)

	<-time.After(1100 * time.Millisecond)
	assert.Equal(t, true, rc.IsTransmissionConfirmed(key), "transmit should be confirmed after logs are read for the key")

	assert.ErrorIs(t, rc.Accept(key), ErrKeyAlreadySet, "key should not be accepted after transmit confirmed and should return an error")
}
