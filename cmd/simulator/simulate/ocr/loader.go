package ocr

import (
	"log"
	"sync"

	ocr2config "github.com/smartcontractkit/libocr/offchainreporting2plus/confighelper"
	"github.com/smartcontractkit/libocr/offchainreporting2plus/ocr3confighelper"
	"github.com/smartcontractkit/libocr/offchainreporting2plus/types"

	"github.com/smartcontractkit/ocr2keepers/cmd/simulator/config"
	"github.com/smartcontractkit/ocr2keepers/cmd/simulator/simulate/chain"
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

// OCR3ConfigLoader ...
type OCR3ConfigLoader struct {
	// provided dependencies
	digest Digester
	logger *log.Logger

	// internal state values
	mu      sync.Mutex
	count   uint64
	oracles []ocr2config.OracleIdentityExtra
	events  map[string]config.ConfigEvent
}

// NewOCR3ConfigLoader ...
func NewOCR3ConfigLoader(rb config.RunBook, digest Digester, logger *log.Logger) *OCR3ConfigLoader {
	eventLookup := make(map[string]config.ConfigEvent)

	for _, event := range rb.ConfigEvents {
		eventLookup[event.Block.String()] = event
	}

	return &OCR3ConfigLoader{
		logger:  log.New(logger.Writer(), "[ocr3-config-loader] ", log.Ldate|log.Ltime|log.Lshortfile),
		digest:  digest,
		events:  eventLookup,
		oracles: []ocr2config.OracleIdentityExtra{},
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

func buildConfig(conf config.ConfigEvent, oracles []ocr2config.OracleIdentityExtra, digester Digester, count uint64) (types.ContractConfig, error) {
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
		conf.F,                      // f int,
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
