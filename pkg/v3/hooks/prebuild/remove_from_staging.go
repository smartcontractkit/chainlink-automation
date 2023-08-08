package prebuild

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/big"

	"github.com/ethereum/go-ethereum/crypto"

	ocr2keepersv3 "github.com/smartcontractkit/ocr2keepers/pkg/v3"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/telemetry"
	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg/v3/types"
)

type resultRemover interface {
	Remove(...string)
}

func NewRemoveFromStaging(remover resultRemover, logger *log.Logger) *RemoveFromStagingHook {
	return &RemoveFromStagingHook{
		remover: remover,
		logger:  log.New(logger.Writer(), fmt.Sprintf("[%s | pre-build hook:remove-from-staging]", telemetry.ServiceName), telemetry.LogPkgStdFlags),
	}
}

type RemoveFromStagingHook struct {
	remover resultRemover
	logger  *log.Logger
}

// UpkeepWorkID returns the identifier using the given upkeepID and trigger extension(tx hash and log index).
func UpkeepWorkID(id *big.Int, trigger ocr2keepers.Trigger) (string, error) {
	extensionBytes, err := json.Marshal(trigger.LogTriggerExtension)
	if err != nil {
		return "", err
	}

	// TODO (auto-4314): Ensure it works with conditionals and add unit tests
	combined := fmt.Sprintf("%s%s", id, extensionBytes)
	hash := crypto.Keccak256([]byte(combined))
	return hex.EncodeToString(hash[:]), nil
}

func (hook *RemoveFromStagingHook) RunHook(outcome ocr2keepersv3.AutomationOutcome) error {
	toRemove := make([]string, 0, len(outcome.Performable))

	for _, result := range outcome.Performable {
		workID, err := UpkeepWorkID(result.UpkeepID.BigInt(), result.Trigger)
		if err != nil {
			continue
		}
		toRemove = append(toRemove, workID)
	}

	hook.logger.Printf("%d results found in outcome", len(toRemove))
	hook.remover.Remove(toRemove...)

	return nil
}
