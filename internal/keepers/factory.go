package keepers

import (
	"fmt"
	"log"
	"runtime"
	"time"

	"github.com/smartcontractkit/libocr/offchainreporting2/types"
	ktypes "github.com/smartcontractkit/ocr2keepers/pkg/types"
)

// NewReportingPluginFactory returns an OCR ReportingPluginFactory. When the plugin
// starts, a separate service is started as a separate go-routine automatically. There
// is no start or stop function for this service so stopping this service relies on
// releasing references to the plugin such that the Go garbage collector cleans up
// hanging routines automatically.
func NewReportingPluginFactory(registry ktypes.Registry, encoder ktypes.ReportEncoder, logger *log.Logger, clearCacheInterval time.Duration) types.ReportingPluginFactory {
	return &keepersReportingFactory{
		registry:   registry,
		encoder:    encoder,
		logger:     logger,
		clearCache: clearCacheInterval,
	}
}

type keepersReportingFactory struct {
	registry   ktypes.Registry
	encoder    ktypes.ReportEncoder
	logger     *log.Logger
	clearCache time.Duration
}

var _ types.ReportingPluginFactory = (*keepersReportingFactory)(nil)

// NewReportingPlugin implements the libocr/offchainreporting2/types ReportingPluginFactory interface
func (d *keepersReportingFactory) NewReportingPlugin(c types.ReportingPluginConfig) (types.ReportingPlugin, types.ReportingPluginInfo, error) {
	var offChainCfg offChainConfig
	err := decode(c.OffchainConfig, &offChainCfg)
	if err != nil {
		return nil, types.ReportingPluginInfo{}, fmt.Errorf("%w: failed to decode off chain config", err)
	}

	info := types.ReportingPluginInfo{
		Name: fmt.Sprintf("Oracle %d: Keepers Plugin Instance w/ Digest '%s'", c.OracleID, c.ConfigDigest),
		Limits: types.ReportingPluginLimits{
			// queries should be empty anyway with the current implementation
			MaxQueryLength: 0,
			// an upkeep key is composed of a block number and upkeep id (~8 bytes)
			// an observation is multiple upkeeps to be performed
			// 100 upkeeps to be performed would be a very high upper limit
			// 100 * 10 = 1_000 bytes
			MaxObservationLength: 1_000,
			// a report is composed of 1 or more abi encoded perform calls
			// with performData of arbitrary length
			MaxReportLength: 10_000, // TODO (config): pick sane limit based on expected performData size. maybe set this to block size limit?
		},
		// unique reports ensures that each round produces only a single report
		UniqueReports: true,
	}

	// TODO (config): cache expiration should be configurable based on offchain
	// config, block time, round time, or other environmental condition
	cacheExpire := 20 * time.Minute

	// TODO (config): number of workers should be based on total amount of resources
	// available. the work load of checking upkeeps is memory heavy as each work
	// item is mostly waiting on the network. many work items get staged very
	// quickly and stay in memory until the network response comes in. from
	// there it's just a matter of decoding the response.
	workers := 10 * runtime.GOMAXPROCS(0) // # of workers = 10 * [# of cpus]

	// TODO (config): the worker queue length should be large enough to accomodate the
	// total number of work items coming in (upkeeps to check per block) without
	// overrunning memory limits.
	workerQueueLength := 1000

	// TODO (config): sample ratio should be able to be calculated based on number of
	// nodes and max number of faulty nodes
	sample := sampleRatio(0.6)

	service := newSimpleUpkeepService(sample, d.registry, d.logger, cacheExpire, d.clearCache, workers, workerQueueLength)

	return &keepers{service: service, encoder: d.encoder}, info, nil
}
