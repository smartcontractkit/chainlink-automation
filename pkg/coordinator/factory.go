package coordinator

import (
	"log"
	"time"

	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg"
	"github.com/smartcontractkit/ocr2keepers/pkg/config"
)

// CoordinatorFactory ...
type CoordinatorFactory struct {
	Logger     *log.Logger
	Encoder    Encoder
	Logs       LogProvider
	CacheClean time.Duration
}

// NewConditionalObserver ...
func (f *CoordinatorFactory) NewCoordinator(c config.OffchainConfig) (ocr2keepers.Coordinator, error) {
	return NewReportCoordinator(
		time.Duration(c.PerformLockoutWindow),
		f.CacheClean,
		f.Logs,
		c.MinConfirmations,
		f.Logger,
		f.Encoder,
	), nil
}
