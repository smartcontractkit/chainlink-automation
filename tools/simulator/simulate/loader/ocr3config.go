package loader

import (
	"log"
	"sync"

	ocr2config "github.com/smartcontractkit/libocr/offchainreporting2plus/confighelper"
	"github.com/smartcontractkit/libocr/offchainreporting2plus/ocr3confighelper"
	"github.com/smartcontractkit/libocr/offchainreporting2plus/types"

	"github.com/smartcontractkit/chainlink-automation/tools/simulator/config"
	"github.com/smartcontractkit/chainlink-automation/tools/simulator/simulate/chain"
)

const (
	ocr3ConfigProgressNamespace = "Emitting OCR3 config transactions"
)

type KeySourcer interface {
	PublicKey() types.OnchainPublicKey
	PKString() string
}

type OffchainKeySourcer interface {
	OffchainPublicKey() types.OffchainPublicKey
	ConfigEncryptionPublicKey() types.ConfigEncryptionPublicKey
}

type Digester interface {
	ConfigDigest(config types.ContractConfig) (types.ConfigDigest, error)
}

type ProgressTelemetry interface {
	Register(string, int64) error
	Increment(string, int64)
}

// OCR3ConfigLoader ...
type OCR3ConfigLoader struct {
	// provided dependencies
	digest   Digester
	progress ProgressTelemetry
	logger   *log.Logger

	// internal state values
	mu      sync.Mutex
	count   uint64
	oracles []ocr2config.OracleIdentityExtra
	events  map[string]config.OCR3ConfigEvent
}

// NewOCR3ConfigLoader adds OCR3 config transactions to incoming blocks as
// defined in a provided simulation plan.
func NewOCR3ConfigLoader(plan config.SimulationPlan, progress ProgressTelemetry, digest Digester, logger *log.Logger) *OCR3ConfigLoader {
	eventLookup := make(map[string]config.OCR3ConfigEvent)

	for _, event := range plan.ConfigEvents {
		eventLookup[event.Event.TriggerBlock.String()] = event
	}

	if progress != nil {
		_ = progress.Register(ocr3ConfigProgressNamespace, int64(len(plan.ConfigEvents)))
	}

	return &OCR3ConfigLoader{
		digest:   digest,
		progress: progress,
		logger:   log.New(logger.Writer(), "[ocr3-config-loader] ", log.Ldate|log.Ltime|log.Lshortfile),
		events:   eventLookup,
		oracles:  []ocr2config.OracleIdentityExtra{},
	}
}

// Load implements the chain.BlockLoaderFunc type and loads ocr config
// transactions into a block on specified block numbers
func (l *OCR3ConfigLoader) Load(block *chain.Block) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// check if new block indicates a new config event should be loaded
	if evt, ok := l.events[block.Number.String()]; ok {
		conf, err := buildConfig(evt, l.oracles, l.digest, l.count+1)
		if err != nil {
			l.logger.Printf("error building config: %s", err)

			return
		}

		l.logger.Printf("config loaded at %s with %d oracles", block.Number, len(l.oracles))

		block.Transactions = append(block.Transactions, chain.OCR3ConfigTransaction{
			Config: conf,
		})

		if l.progress != nil {
			l.progress.Increment(ocr3ConfigProgressNamespace, 1)
		}

		l.count++
	}
}

func (l *OCR3ConfigLoader) AddSigner(id string, onKey KeySourcer, offKey OffchainKeySourcer) {
	l.mu.Lock()
	defer l.mu.Unlock()

	newOracle := ocr2config.OracleIdentityExtra{
		OracleIdentity: ocr2config.OracleIdentity{
			OffchainPublicKey: offKey.OffchainPublicKey(),
			OnchainPublicKey:  onKey.PublicKey(),
			PeerID:            id,
			TransmitAccount:   types.Account(onKey.PKString()),
		},
		ConfigEncryptionPublicKey: offKey.ConfigEncryptionPublicKey(),
	}

	l.oracles = append(l.oracles, newOracle)
}

func buildConfig(conf config.OCR3ConfigEvent, oracles []ocr2config.OracleIdentityExtra, digester Digester, count uint64) (types.ContractConfig, error) {
	// S is a slice of values that indicate the number of oracles involved
	// in attempting to transmit. For the simulator, all nodes will be involved
	// in transmit attempts.
	S := make([]int, len(oracles))
	for i := 0; i < len(oracles); i++ {
		S[i] = 1
	}

	signerOnchainPublicKeys, transmitterAccounts, f, onchainConfig, offchainConfigVersion, offchainConfig, err := ocr3confighelper.ContractSetConfigArgsForTests(
		conf.DeltaProgress.Value(),  // deltaProgress time.Duratioonfn,
		conf.DeltaResend.Value(),    // deltaResend time.Duration,
		conf.DeltaInitial.Value(),   // deltaInitial time.Duration
		conf.DeltaRound.Value(),     // deltaRound time.Duration,
		conf.DeltaGrace.Value(),     // deltaGrace time.Duration,
		conf.DeltaRequest.Value(),   // deltaCertifiedCommitRequest time.Duration
		conf.DeltaStage.Value(),     // deltaStage time.Duration
		conf.Rmax,                   // rMax uint64
		S,                           // s []int,
		oracles,                     // oracles []OracleIdentityExtra,
		[]byte(conf.Offchain),       // reportingPluginConfig []byte,
		conf.MaxQuery.Value(),       // maxDurationQuery time.Duration,
		conf.MaxObservation.Value(), // maxDurationObservation time.Duration,
		conf.MaxAccept.Value(),      // maxDurationShouldAcceptAttestedReport time.Duration
		conf.MaxTransmit.Value(),    // maxDurationShouldTransmitAcceptedReport time.Duration,
		conf.MaxFaultyNodesF,        // f int,
		nil,                         // onchainConfig []byte,
	)
	if err != nil {
		return types.ContractConfig{}, err
	}

	contractConf := types.ContractConfig{
		ConfigCount:           uint64(count),
		Signers:               signerOnchainPublicKeys,
		Transmitters:          transmitterAccounts,
		F:                     f,
		OnchainConfig:         onchainConfig,
		OffchainConfigVersion: offchainConfigVersion,
		OffchainConfig:        offchainConfig,
	}

	digest, _ := digester.ConfigDigest(contractConf)
	contractConf.ConfigDigest = digest

	return contractConf, nil
}
