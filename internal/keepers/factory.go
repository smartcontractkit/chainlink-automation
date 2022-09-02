package keepers

import (
	"fmt"

	"github.com/smartcontractkit/libocr/offchainreporting2/types"
	ktypes "github.com/smartcontractkit/ocr2keepers/pkg/types"
)

func NewReportingPluginFactory(registry ktypes.Registry) types.ReportingPluginFactory {
	return &keepersReportingFactory{registry: registry}
}

type keepersReportingFactory struct {
	registry ktypes.Registry
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

	service := NewSimpleUpkeepService(SampleRatio(0.3), d.registry)

	return &keepers{service: service}, info, nil
}
