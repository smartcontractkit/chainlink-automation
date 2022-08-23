package keepers

import (
	"fmt"

	"github.com/smartcontractkit/libocr/offchainreporting2/types"
)

func NewReportingPluginFactory() types.ReportingPluginFactory {
	return nil
}

type keepersReportingFactory struct{}

var _ types.ReportingPluginFactory = (*keepersReportingFactory)(nil)

func (d *keepersReportingFactory) NewReportingPlugin(c types.ReportingPluginConfig) (types.ReportingPlugin, types.ReportingPluginInfo, error) {
	info := types.ReportingPluginInfo{
		Name: fmt.Sprintf("keepers instance %v", "TODO: give instance a unique name"),
		Limits: types.ReportingPluginLimits{
			MaxQueryLength:       1000,      // TODO: pick sane limits
			MaxObservationLength: 1_000_000, // TODO: pick sane limits
			MaxReportLength:      10_000,    // TODO: pick sane limits
		},
		UniqueReports: true, // TODO: pick sane limits
	}

	return &keepers{}, info, nil
}
