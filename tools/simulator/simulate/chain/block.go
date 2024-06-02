package chain

import (
	"fmt"
	"math/big"
	"strconv"
	"sync"

	"github.com/smartcontractkit/libocr/offchainreporting2plus/types"
)

type Block struct {
	Hash         [32]byte
	Number       *big.Int
	Transactions []interface{}
}

type Log struct {
	TxHash       [32]byte
	BlockNumber  *big.Int
	BlockHash    [32]byte
	Idx          uint32
	TriggerValue string
}

type OCR3ConfigTransaction struct {
	Config types.ContractConfig
}

type PerformUpkeepTransaction struct {
	Transmits []TransmitEvent
}

type UpkeepCreatedTransaction struct {
	Upkeep SimulatedUpkeep
}

// below this line should not be in this package
type UpkeepType int

const (
	ConditionalType UpkeepType = iota
	LogTriggerType
)

type SimulatedUpkeep struct {
	ID             *big.Int
	CreateInBlock  *big.Int
	UpkeepID       [32]byte
	Type           UpkeepType
	AlwaysEligible bool
	EligibleAt     []*big.Int
	TriggeredBy    string
	CheckData      []byte
	Expected       bool
	Retryable      bool
	States         *CheckPipelineStateManager
}

type CheckPipelineStateManager struct {
	nextPosition int
	states       []int
	mu           sync.Mutex
}

func NewCheckPipelineStateManager(pattern string) (*CheckPipelineStateManager, error) {
	states := make([]int, 0, len(pattern))

	for _, rne := range pattern {
		flag, err := strconv.Atoi(string(rne))
		if err != nil {
			return nil, err
		}

		if flag > 1 || flag < 0 {
			return nil, fmt.Errorf("only 0 and 1 allowed")
		}

		states = append(states, flag)
	}

	return &CheckPipelineStateManager{
		states: states,
	}, nil
}

func (m *CheckPipelineStateManager) GetNextState() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.states) == 0 {
		return 0
	}

	nextState := m.states[m.nextPosition]

	m.nextPosition++

	if m.nextPosition >= len(m.states) {
		m.nextPosition = 0
	}

	return nextState
}

type SimulatedLog struct {
	TriggerAt    *big.Int
	TriggerValue string
}

type TransmitEvent struct {
	SendingAddress string
	Report         []byte
	Hash           [32]byte
	Round          uint64
	BlockNumber    *big.Int
	BlockHash      [32]byte
}
