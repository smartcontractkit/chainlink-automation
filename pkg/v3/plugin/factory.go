package plugin

import (
	"fmt"
	"log"
	"math"
	"math/cmplx"
	"strconv"

	"github.com/smartcontractkit/libocr/offchainreporting2plus/ocr3types"

	"github.com/smartcontractkit/ocr2keepers/pkg/v3/config"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/runner"
	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg/v3/types"
)

const (
	// TODO: Calculate these properly, add MaxOutcomeLength
	// MaxObservationLength applies a limit to the total length of bytes in an
	// observation. Observations contain check results, proposals and block
	// coordination data upto a certain number of elements which defines this limit
	MaxObservationLength = 1_000_000
	// MaxOutcomeLength applies a limit to the total length of bytes in an outcome for
	// a round. Outcome contains agreed performables and surfaced proposals upto a
	// certain number of elements which defines this limit
	MaxOutcomeLength = 1_000_000
	// MaxReportLength limits the total length of bytes for a single report. A
	// report is composed of 1 or more abi encoded perform calls with
	// performData of arbitrary length. Reports are limited by gas used to
	// transmit the report, so the length in bytes should be relative to this.
	MaxReportLength = 100_000
	// MaxReportCount limits the total number of reports allowed to be produced
	// by the OCR protocol in a single round. This should be atleast the number
	// of allowed agreed performables in a single round.
	MaxReportCount = 100
)

type pluginFactory struct {
	logProvider        ocr2keepers.LogEventProvider
	events             ocr2keepers.TransmitEventProvider
	blocks             ocr2keepers.BlockSubscriber
	rp                 ocr2keepers.RecoverableProvider
	builder            ocr2keepers.PayloadBuilder
	getter             ocr2keepers.ConditionalUpkeepProvider
	runnable           ocr2keepers.Runnable
	runnerConf         runner.RunnerConfig
	encoder            ocr2keepers.Encoder
	upkeepTypeGetter   ocr2keepers.UpkeepTypeGetter
	upkeepStateUpdater ocr2keepers.UpkeepStateUpdater
	logger             *log.Logger
}

func NewReportingPluginFactory(
	logProvider ocr2keepers.LogEventProvider,
	events ocr2keepers.TransmitEventProvider,
	blocks ocr2keepers.BlockSubscriber,
	rp ocr2keepers.RecoverableProvider,
	builder ocr2keepers.PayloadBuilder,
	getter ocr2keepers.ConditionalUpkeepProvider,
	runnable ocr2keepers.Runnable,
	runnerConf runner.RunnerConfig,
	encoder ocr2keepers.Encoder,
	upkeepTypeGetter ocr2keepers.UpkeepTypeGetter,
	upkeepStateUpdater ocr2keepers.UpkeepStateUpdater,
	logger *log.Logger,
) ocr3types.ReportingPluginFactory[AutomationReportInfo] {
	return &pluginFactory{
		logProvider:        logProvider,
		events:             events,
		blocks:             blocks,
		rp:                 rp,
		builder:            builder,
		getter:             getter,
		runnable:           runnable,
		runnerConf:         runnerConf,
		encoder:            encoder,
		upkeepTypeGetter:   upkeepTypeGetter,
		upkeepStateUpdater: upkeepStateUpdater,
		logger:             logger,
	}
}

func (factory *pluginFactory) NewReportingPlugin(c ocr3types.ReportingPluginConfig) (ocr3types.ReportingPlugin[AutomationReportInfo], ocr3types.ReportingPluginInfo, error) {
	info := ocr3types.ReportingPluginInfo{
		Name: fmt.Sprintf("Oracle: %d: Automation Plugin Instance w/ Digest '%s'", c.OracleID, c.ConfigDigest),
		Limits: ocr3types.ReportingPluginLimits{
			MaxQueryLength:       0,
			MaxObservationLength: MaxObservationLength,
			MaxOutcomeLength:     MaxOutcomeLength,
			MaxReportLength:      MaxReportLength,
			MaxReportCount:       MaxReportCount,
		},
	}

	// decode the off-chain config
	conf, err := config.DecodeOffchainConfig(c.OffchainConfig)
	if err != nil {
		return nil, info, err
	}

	parsed, err := strconv.ParseFloat(conf.TargetProbability, 32)
	if err != nil {
		return nil, info, fmt.Errorf("%w: failed to parse configured probability", err)
	}

	sample, err := sampleFromProbability(conf.TargetInRounds, c.N-c.F, float32(parsed))
	if err != nil {
		return nil, info, fmt.Errorf("%w: failed to create plugin", err)
	}

	// create the plugin; all services start automatically
	p, err := newPlugin(
		c.ConfigDigest,
		factory.logProvider,
		factory.events,
		factory.blocks,
		factory.rp,
		factory.builder,
		sample,
		factory.getter,
		factory.encoder,
		factory.upkeepTypeGetter,
		factory.upkeepStateUpdater,
		factory.runnable,
		factory.runnerConf,
		conf,
		c.F,
		factory.logger,
	)
	if err != nil {
		return nil, info, err
	}

	return p, info, nil
}

func sampleFromProbability(rounds, nodes int, probability float32) (sampleRatio, error) {
	var ratio sampleRatio

	if rounds <= 0 {
		return ratio, fmt.Errorf("number of rounds must be greater than 0")
	}

	if nodes <= 0 {
		return ratio, fmt.Errorf("number of nodes must be greater than 0")
	}

	if probability > 1 || probability <= 0 {
		return ratio, fmt.Errorf("probability must be less than 1 and greater than 0")
	}

	r := complex(float64(rounds), 0)
	n := complex(float64(nodes), 0)
	p := complex(float64(probability), 0)

	// calculate the probability that x of total selection collectively will
	// cover all of a selection by all nodes over number of rounds
	g := -1.0 * (p - 1.0)
	x := cmplx.Pow(cmplx.Pow(g, 1.0/r), 1.0/n)
	rat := cmplx.Abs(-1.0 * (x - 1.0))
	rat = math.Round(rat/0.01) * 0.01
	ratio = sampleRatio(float32(rat))

	return ratio, nil
}

type sampleRatio float32

func (r sampleRatio) OfInt(count int) int {
	// rounds the result using basic rounding op
	return int(math.Round(float64(r) * float64(count)))
}

func (r sampleRatio) String() string {
	return fmt.Sprintf("%.8f", float32(r))
}
