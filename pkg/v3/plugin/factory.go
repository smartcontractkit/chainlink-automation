package plugin

import (
	"fmt"

	"github.com/smartcontractkit/libocr/offchainreporting2plus/ocr3types"
	"github.com/smartcontractkit/ocr2keepers/pkg/config"
)

const (
	// MaxObservationLength applies a limit to the total length of bytes in an
	// observation. Observations can become quite large due to multiple
	// CheckResult objects and block coordination data. This is set to 1MB for
	// now but might either need to be increased or data compression be applied.
	MaxObservationLength = 1_000_000
	// MaxReportLength limits the total length of bytes for a single report. A
	// report is composed of 1 or more abi encoded perform calls with
	// performData of arbitrary length. Reports are limited by gas usaged to
	// transmit the report, so the length in bytes should be relative to this.
	MaxReportLength = 10_000
	// MaxReportCount limits the total number of reports allowed to be produced
	// by the OCR protocol. Limiting to a high number for now because each
	// report will be signed independently.
	MaxReportCount = 20
)

type pluginFactory[RI any] struct{}

func NewReportingPluginFactory[RI any]() ocr3types.OCR3PluginFactory[RI] {
	return &pluginFactory[RI]{}
}

func (factory *pluginFactory[RI]) NewOCR3Plugin(c ocr3types.OCR3PluginConfig) (ocr3types.OCR3Plugin[RI], ocr3types.OCR3PluginInfo, error) {
	info := ocr3types.OCR3PluginInfo{
		Name: fmt.Sprintf("Oracle: %d: Automation Plugin Instance w/ Digest '%s'", c.OracleID, c.ConfigDigest),
		Limits: ocr3types.OCR3PluginLimits{
			MaxQueryLength:       0,
			MaxObservationLength: MaxObservationLength,
			MaxOutcomeLength:     MaxObservationLength, // outcome length can be the same as observation length
			MaxReportLength:      MaxReportLength,
			MaxReportCount:       MaxReportCount,
		},
	}

	// decode the off-chain config
	_, err := config.DecodeOffchainConfig(c.OffchainConfig)
	if err != nil {
		return nil, info, err
	}

	// create the plugin; all services start automatically
	p, err := newPlugin[RI](nil, nil)
	if err != nil {
		return nil, info, err
	}

	return p, info, nil
}
