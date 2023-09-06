package db

import (
	"context"
	"fmt"
	"sync"

	"github.com/smartcontractkit/libocr/offchainreporting2plus/types"
)

type SimulatedOCR3Database struct {
	mu                   sync.RWMutex
	states               map[[32]byte]types.PersistentState
	transmitDigestLookup map[[32]byte][]types.ReportTimestamp
	pendingTransmit      map[types.ReportTimestamp]types.PendingTransmission

	// used values
	protoState map[string][]byte
	config     *types.ContractConfig
}

func NewSimulatedOCR3Database() *SimulatedOCR3Database {
	return &SimulatedOCR3Database{
		states:               make(map[[32]byte]types.PersistentState),
		transmitDigestLookup: make(map[[32]byte][]types.ReportTimestamp),
		pendingTransmit:      make(map[types.ReportTimestamp]types.PendingTransmission),
		// used values
		protoState: make(map[string][]byte),
	}
}

func (d *SimulatedOCR3Database) ReadConfig(_ context.Context) (*types.ContractConfig, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.config == nil {
		return nil, fmt.Errorf("not found")
	}

	return d.config, nil
}

func (d *SimulatedOCR3Database) WriteConfig(_ context.Context, config types.ContractConfig) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.config = &config
	return nil
}

// In case the key is not found, nil should be returned.
func (d *SimulatedOCR3Database) ReadProtocolState(ctx context.Context, configDigest types.ConfigDigest, key string) ([]byte, error) {
	// might need to check against latest config digest or scope to digest
	val, ok := d.protoState[key]
	if !ok {
		return nil, fmt.Errorf("state not found for key: %s", key)
	}

	return val, nil
}

// Writing with a nil value is the same as deleting.
func (d *SimulatedOCR3Database) WriteProtocolState(ctx context.Context, configDigest types.ConfigDigest, key string, value []byte) error {
	d.protoState[key] = value

	// might need to check against latest config digest or scope to digest
	return nil
}
