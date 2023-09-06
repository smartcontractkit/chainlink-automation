package ocr

import (
	"context"
	"fmt"
	"log"
	"runtime"
	"sync"

	"github.com/smartcontractkit/libocr/offchainreporting2plus/types"

	"github.com/smartcontractkit/ocr2keepers/cmd/simv3/simulator/chain"
)

// OCR3ConfigTracker ...
type OCR3ConfigTracker struct {
	// provided dependencies
	listener *chain.Listener
	logger   *log.Logger

	// internal state props
	mu       sync.RWMutex
	block    uint64
	config   *types.ContractConfig
	chNotify chan struct{}

	// service values
	chDone chan struct{}
}

// NewOCR3ConfigTracker ...
func NewOCR3ConfigTracker(listener *chain.Listener, logger *log.Logger) *OCR3ConfigTracker {
	src := &OCR3ConfigTracker{
		listener: listener,
		logger:   log.New(logger.Writer(), "[ocr3-config-tracker]", log.LstdFlags),
		chNotify: make(chan struct{}),
		chDone:   make(chan struct{}),
	}

	go src.run()

	runtime.SetFinalizer(src, func(srv *OCR3ConfigTracker) { srv.stop() })

	return src
}

// Notify may optionally emit notification events when the contract's
// configuration changes. This is purely used as an optimization reducing
// the delay between a configuration change and its enactment. Implementors
// who don't care about this may simply return a nil channel.
//
// The returned channel should never be closed.
func (ct *OCR3ConfigTracker) Notify() <-chan struct{} {
	return ct.chNotify
}

// LatestConfigDetails returns information about the latest configuration,
// but not the configuration itself.
func (ct *OCR3ConfigTracker) LatestConfigDetails(_ context.Context) (changedInBlock uint64, configDigest types.ConfigDigest, err error) {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	if ct.config == nil {
		return 0, types.ConfigDigest{}, fmt.Errorf("no config found")
	}

	return ct.block, ct.config.ConfigDigest, nil
}

// LatestConfig returns the latest configuration.
func (ct *OCR3ConfigTracker) LatestConfig(_ context.Context, _ uint64) (types.ContractConfig, error) {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	if ct.config == nil {
		return types.ContractConfig{}, fmt.Errorf("no config found")
	}

	return *ct.config, nil
}

// LatestBlockHeight returns the height of the most recent block in the chain.
func (ct *OCR3ConfigTracker) LatestBlockHeight(ctx context.Context) (blockHeight uint64, err error) {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	if ct.config == nil {
		return 0, fmt.Errorf("no config found")
	}

	return ct.block, nil
}

func (ct *OCR3ConfigTracker) run() {
	chEvents := ct.listener.Subscribe(chain.OCR3ConfigChannel)

	for {
		select {
		case event := <-chEvents:
			switch evt := event.Event.(type) {
			case chain.OCR3ConfigTransaction:
				ct.setConfig(event.BlockNumber.Uint64(), evt.Config)
			}
		case <-ct.chDone:
			return
		}
	}
}

func (ct *OCR3ConfigTracker) setConfig(block uint64, config types.ContractConfig) {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	ct.block = block
	ct.config = &config

	go func() { ct.chNotify <- struct{}{} }()
}

func (ct *OCR3ConfigTracker) stop() {
	close(ct.chDone)
}
