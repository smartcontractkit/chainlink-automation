package simulators

import (
	"context"
	"fmt"
	"sync"

	"github.com/smartcontractkit/libocr/offchainreporting2plus/types"
)

type SimulatedDatabase struct {
	mu                   sync.RWMutex
	states               map[[32]byte]types.PersistentState
	transmitDigestLookup map[[32]byte][]types.ReportTimestamp
	pendingTransmit      map[types.ReportTimestamp]types.PendingTransmission

	// used values
	protoState map[string][]byte
	config     *types.ContractConfig
}

func NewSimulatedDatabase() *SimulatedDatabase {
	return &SimulatedDatabase{
		states:               make(map[[32]byte]types.PersistentState),
		transmitDigestLookup: make(map[[32]byte][]types.ReportTimestamp),
		pendingTransmit:      make(map[types.ReportTimestamp]types.PendingTransmission),
		// used values
		protoState: make(map[string][]byte),
	}
}

func (d *SimulatedDatabase) ReadConfig(_ context.Context) (*types.ContractConfig, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.config == nil {
		return nil, fmt.Errorf("not found")
	}

	return d.config, nil
}

func (d *SimulatedDatabase) WriteConfig(_ context.Context, config types.ContractConfig) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.config = &config
	return nil
}

/*
func (d *SimulatedDatabase) ReadState(_ context.Context, configDigest types.ConfigDigest) (*types.PersistentState, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	val, ok := d.states[configDigest]
	if ok {
		return &val, nil
	}
	return nil, fmt.Errorf("not found")
}

func (d *SimulatedDatabase) WriteState(_ context.Context, configDigest types.ConfigDigest, state types.PersistentState) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.states[configDigest] = state

	return nil
}

func (d *SimulatedDatabase) StorePendingTransmission(_ context.Context, ts types.ReportTimestamp, tr types.PendingTransmission) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.pendingTransmit[ts] = tr
	return nil
}

func (d *SimulatedDatabase) PendingTransmissionsWithConfigDigest(_ context.Context, digest types.ConfigDigest) (map[types.ReportTimestamp]types.PendingTransmission, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	result := make(map[types.ReportTimestamp]types.PendingTransmission)
	keys, ok := d.transmitDigestLookup[digest]
	if ok {
		for _, key := range keys {
			value, ok := d.pendingTransmit[key]
			if ok {
				result[key] = value
			}
		}
	}

	return result, nil
}

func (d *SimulatedDatabase) DeletePendingTransmission(_ context.Context, ts types.ReportTimestamp) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	delete(d.pendingTransmit, ts)

	return nil
}

func (d *SimulatedDatabase) DeletePendingTransmissionsOlderThan(ctx context.Context, tm time.Time) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	toDelete := make([]types.ReportTimestamp, 0)

	for key, value := range d.pendingTransmit {
		if value.Time.Before(tm) {
			toDelete = append(toDelete, key)
		}
	}

	for _, key := range toDelete {
		delete(d.pendingTransmit, key)
	}

	// TODO: should also delete from lookup, but it's not critical

	return nil
}
*/

// In case the key is not found, nil should be returned.
func (d *SimulatedDatabase) ReadProtocolState(ctx context.Context, configDigest types.ConfigDigest, key string) ([]byte, error) {
	// might need to check against latest config digest or scope to digest
	val, ok := d.protoState[key]
	if !ok {
		return nil, fmt.Errorf("state not found for key: %s", key)
	}

	return val, nil
}

// Writing with a nil value is the same as deleting.
func (d *SimulatedDatabase) WriteProtocolState(ctx context.Context, configDigest types.ConfigDigest, key string, value []byte) error {
	d.protoState[key] = value

	// might need to check against latest config digest or scope to digest
	return nil
}
