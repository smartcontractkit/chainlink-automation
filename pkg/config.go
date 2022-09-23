package ocr2keepers

import (
	"time"

	"github.com/smartcontractkit/libocr/commontypes"
	"github.com/smartcontractkit/libocr/offchainreporting2/types"
	ktypes "github.com/smartcontractkit/ocr2keepers/pkg/types"
)

const (
	// DefaultCacheClearInterval is the default setting for the interval at
	// which the cache attempts to evict expired keys
	DefaultCacheClearInterval = 30 * time.Second
)

// DelegateConfig provides a single configuration struct for all options
// to be passed to the oracle, oracle factory, and underlying plugin/services.
type DelegateConfig struct {
	BinaryNetworkEndpointFactory types.BinaryNetworkEndpointFactory
	V2Bootstrappers              []commontypes.BootstrapperLocator
	ContractConfigTracker        types.ContractConfigTracker
	ContractTransmitter          types.ContractTransmitter
	KeepersDatabase              types.Database
	Logger                       commontypes.Logger
	MonitoringEndpoint           commontypes.MonitoringEndpoint
	OffchainConfigDigester       types.OffchainConfigDigester
	OffchainKeyring              types.OffchainKeyring
	OnchainKeyring               types.OnchainKeyring
	LocalConfig                  types.LocalConfig

	// Registry is an abstract plugin registry; can be evm based or anything else
	Registry ktypes.Registry
	// ReportEncoder is an abstract encoder for encoding reports destined for trasmission; can be evm based or anything else
	ReportEncoder ktypes.ReportEncoder
	// ClearCacheInterval is a configural parameter for how often the cache attempts to evict expired keys
	ClearCacheInterval time.Duration
}
