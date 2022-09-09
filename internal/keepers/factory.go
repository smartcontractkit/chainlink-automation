package keepers

import (
	"fmt"
	"log"

	"github.com/smartcontractkit/libocr/offchainreporting2/types"
	ktypes "github.com/smartcontractkit/ocr2keepers/pkg/types"
)

func NewReportingPluginFactory(registry ktypes.Registry, encoder ktypes.ReportEncoder, logger *log.Logger) types.ReportingPluginFactory {
	return &keepersReportingFactory{registry: registry, encoder: encoder, logger: logger}
}

type keepersReportingFactory struct {
	registry ktypes.Registry
	encoder  ktypes.ReportEncoder
	logger   *log.Logger
}

var _ types.ReportingPluginFactory = (*keepersReportingFactory)(nil)

// NewReportingPlugin implements the libocr/offchainreporting2/types ReportingPluginFactory interface
func (d *keepersReportingFactory) NewReportingPlugin(c types.ReportingPluginConfig) (types.ReportingPlugin, types.ReportingPluginInfo, error) {
	info := types.ReportingPluginInfo{
		Name: fmt.Sprintf("keepers instance %v", "TODO: give instance a unique name"),
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
			MaxReportLength: 10_000, // TODO: pick sane limit
		},
		// unique reports ensures that each round produces only a single report
		UniqueReports: true,
	}

	// set default logger to write to debug logs
	// set up log formats
	//log.SetOutput(d.logger.Writer())
	//log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile | log.LUTC)

	service := NewSimpleUpkeepService(SampleRatio(0.01), d.registry, d.logger)

	return &keepers{service: service, encoder: d.encoder}, info, nil
}
