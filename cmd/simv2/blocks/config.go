package blocks

import (
	"log"
	"sync"
	"time"

	ocr2config "github.com/smartcontractkit/libocr/offchainreporting2/confighelper"

	"github.com/smartcontractkit/libocr/offchainreporting2/types"
	"github.com/smartcontractkit/ocr2keepers/cmd/simv2/config"
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

type ConfigLoader struct {
	mu              sync.Mutex
	loadNextBlock   bool
	oracles         []ocr2config.OracleIdentityExtra
	lastConfigEvent *config.ConfigEvent
	events          map[string]config.ConfigEvent
	digest          Digester
	count           int
}

func NewConfigLoader(events []config.ConfigEvent, digest Digester) *ConfigLoader {
	eventLookup := make(map[string]config.ConfigEvent)

	for _, event := range events {
		eventLookup[event.Block.String()] = event
	}

	return &ConfigLoader{
		events:  eventLookup,
		oracles: []ocr2config.OracleIdentityExtra{},
		digest:  digest,
	}
}

func (l *ConfigLoader) Load(block *config.SymBlock) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// check if new block indicates a new config event should be loaded
	evt, ok := l.events[block.BlockNumber.String()]
	if ok {
		l.lastConfigEvent = &evt
		l.count++
		l.loadNextBlock = true
	}

	// skip if not transmitting or if no config is loaded
	if !l.loadNextBlock || l.lastConfigEvent == nil {
		return
	}

	log.Printf("number of oracles in loaded config: %d", len(l.oracles))
	// load the block with a ContractConfig
	conf, err := buildConfig(*l.lastConfigEvent, l.oracles, l.digest, l.count)
	if err != nil {
		log.Printf("error building config: %s", err)
	}

	block.Config = &conf
	l.loadNextBlock = false
}

func (l *ConfigLoader) AddSigner(id string, onKey KeySourcer, offKey OffchainKeySourcer) {
	l.mu.Lock()
	defer l.mu.Unlock()

	newOracle := ocr2config.OracleIdentityExtra{
		OracleIdentity: ocr2config.OracleIdentity{
			OnchainPublicKey:  onKey.PublicKey(),
			OffchainPublicKey: offKey.OffchainPublicKey(),
			PeerID:            id,
			TransmitAccount:   types.Account(onKey.PKString()),
		},
		ConfigEncryptionPublicKey: offKey.ConfigEncryptionPublicKey(),
	}

	l.oracles = append(l.oracles, newOracle)
}

func buildConfig(c config.ConfigEvent, oracles []ocr2config.OracleIdentityExtra, digester Digester, count int) (types.ContractConfig, error) {
	S := make([]int, len(oracles))
	for i := 0; i < len(oracles); i++ {
		S[i] = 1
	}

	signerOnchainPublicKeys, transmitterAccounts, f, onchainConfig, offchainConfigVersion, offchainConfig, err := ocr2config.ContractSetConfigArgsForTests(
		c.DeltaProgress.Value(),  // deltaProgress time.Duration,
		c.DeltaResend.Value(),    // deltaResend time.Duration,
		c.DeltaRound.Value(),     // deltaRound time.Duration,
		c.DeltaGrace.Value(),     // deltaGrace time.Duration,
		c.DeltaStage.Value(),     // deltaStage time.Duration,
		c.Rmax,                   // rMax uint8,
		S,                        // s []int,
		oracles,                  // oracles []OracleIdentityExtra,
		c.Offchain,               // reportingPluginConfig []byte,
		50*time.Millisecond,      // maxDurationQuery time.Duration,
		c.MaxObservation.Value(), // maxDurationObservation time.Duration,
		c.MaxReport.Value(),      // maxDurationReport time.Duration,
		c.MaxAccept.Value(),      // maxDurationShouldAcceptFinalizedReport time.Duration,
		c.MaxTransmit.Value(),    // maxDurationShouldTransmitAcceptedReport time.Duration,
		c.F,                      // f int,
		nil,                      // onchainConfig []byte,
	)
	if err != nil {
		return types.ContractConfig{}, err
	}

	conf := types.ContractConfig{
		ConfigCount:           uint64(count),
		Signers:               signerOnchainPublicKeys,
		Transmitters:          transmitterAccounts,
		F:                     f,
		OnchainConfig:         onchainConfig,
		OffchainConfigVersion: offchainConfigVersion,
		OffchainConfig:        offchainConfig,
	}

	digest, _ := digester.ConfigDigest(conf)
	conf.ConfigDigest = digest

	return conf, nil
}
