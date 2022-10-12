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

	key := types.UpkeepKey([]byte("1|1"))
	id := types.UpkeepIdentifier([]byte("1"))
	mr.Mock.On("IdentifierFromKey", key).Return(id, nil).Times(6)

	f := rc.Filter()
	assert.Equal(t, true, f(key))
	assert.Equal(t, true, rc.IsTransmissionConfirmed(key))

	assert.NoError(t, rc.Accept(key))
	assert.Equal(t, false, f(key))
	assert.Equal(t, false, rc.IsTransmissionConfirmed(key))

	mp.Mock.On("PerformLogs", mock.Anything).Return([]types.PerformLog{
		{Key: key},
	}, nil)

	<-time.After(1100 * time.Millisecond)
	assert.Equal(t, true, rc.IsTransmissionConfirmed(key))
}
