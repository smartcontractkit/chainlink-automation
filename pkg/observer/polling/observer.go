package polling

import (
	"context"
	"fmt"
	"log"
	"runtime"
	"sync"
	"time"

	"github.com/smartcontractkit/ocr2keepers/internal/util"
	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg"
	"github.com/smartcontractkit/ocr2keepers/pkg/observer"
)

var ErrTooManyErrors = fmt.Errorf("too many errors in parallel worker process")

type UpkeepProvider interface {
	GetActiveUpkeepIDs(context.Context) ([]ocr2keepers.UpkeepIdentifier, error)
}

type Encoder interface {
	// MakeUpkeepKey combines a block and upkeep id into an upkeep key. This
	// will probably go away with a more structured static upkeep type.
	MakeUpkeepKey(ocr2keepers.BlockKey, ocr2keepers.UpkeepIdentifier) ocr2keepers.UpkeepKey
	// SplitUpkeepKey ...
	SplitUpkeepKey(ocr2keepers.UpkeepKey) (ocr2keepers.BlockKey, ocr2keepers.UpkeepIdentifier, error)
	// Detail is a temporary value that provides upkeep key and gas to perform.
	// A better approach might be needed here.
	Detail(ocr2keepers.UpkeepResult) (ocr2keepers.UpkeepKey, uint32, error)
}

type Executer interface {
	CheckUpkeep(context.Context, bool, ...ocr2keepers.UpkeepKey) ([]ocr2keepers.UpkeepResult, error)
}

type HeadProvider interface {
	HeadTicker() chan ocr2keepers.BlockKey
}

type Service interface {
	Start()
	Stop()
}

type Shuffler[T any] interface {
	Shuffle([]T) []T
}

// Ratio is an interface that provides functions to calculate a ratio of a given
// input
type Ratio interface {
	// OfInt should return n out of x such that n/x ~ r (ratio)
	OfInt(int) int
	fmt.Stringer
}

func NewPollingObserver(
	logger *log.Logger,
	src UpkeepProvider,
	heads HeadProvider,
	exe Executer,
	enc Encoder,
	ratio Ratio,
	maxSamplingDuration time.Duration, // maximum amount of time allowed for RPC calls per head
	coord ocr2keepers.Coordinator,
	mercuryLookup bool,
) *PollingObserver {
	ctx, cancel := context.WithCancel(context.Background())

	ob := &PollingObserver{
		ctx:              ctx,
		cancel:           cancel,
		logger:           logger,
		samplingDuration: maxSamplingDuration,
		shuffler:         util.Shuffler[ocr2keepers.UpkeepKey]{Source: util.NewCryptoRandSource()}, // use crypto/rand shuffling for true random
		ratio:            ratio,
		stager:           &stager{},
		coordinator:      coord,
		src:              src,
		exe:              exe,
		enc:              enc,
		heads:            heads,
		mercuryLookup:    mercuryLookup,
	}

	// make all go-routines started by this entity automatically recoverable
	// on panics
	ob.services = []Service{
		util.NewRecoverableService(&observer.SimpleService{F: ob.runHeadTasks, C: cancel}, logger),
	}

	// automatically stop all services if the reference is no longer reachable
	// this is a safety in the case Stop isn't called explicitly
	runtime.SetFinalizer(ob, func(srv *PollingObserver) { _ = srv.Close() })

	ob.Start()

	return ob
}

type PollingObserver struct {
	ctx       context.Context
	cancel    context.CancelFunc
	startOnce sync.Once
	stopOnce  sync.Once

	// static values provided by constructor
	samplingDuration time.Duration // limits time spent processing a single block
	ratio            Ratio         // ratio for limiting sample size

	// initialized components inside a constructor
	services []Service
	stager   *stager

	// dependency interfaces required by the polling observer
	logger      *log.Logger
	heads       HeadProvider                    // provides new blocks to be operated on
	coordinator ocr2keepers.Coordinator         // key status coordinator tracks in-flight status
	shuffler    Shuffler[ocr2keepers.UpkeepKey] // provides shuffling logic for upkeep keys

	src           UpkeepProvider
	exe           Executer
	enc           Encoder
	mercuryLookup bool
}

// Observe implements the Observer interface and provides a slice of identifiers
// that were observed to be performable along with the block at which they were
// observed. All ids that are pending are filtered out.
func (o *PollingObserver) Observe() (ocr2keepers.BlockKey, []ocr2keepers.UpkeepIdentifier, error) {
	bl, ids := o.stager.get()
	filteredIDs := make([]ocr2keepers.UpkeepIdentifier, 0, len(ids))

	for _, id := range ids {
		key := o.enc.MakeUpkeepKey(bl, id)

		if pending, err := o.coordinator.IsPending(key); pending || err != nil {
			if err != nil {
				o.logger.Printf("error checking pending state for '%s': %s", key, err)
			} else {
				o.logger.Printf("filtered out key '%s'", key)
			}

			continue
		}

		filteredIDs = append(filteredIDs, id)
	}

	return bl, filteredIDs, nil
}

// Start will start all required internal services. Calling this function again
// after the first is a noop.
func (o *PollingObserver) Start() {
	o.startOnce.Do(func() {
		go o.cacheCleaner.Run(o.cache)
		for _, svc := range o.services {
			o.logger.Printf("PollingObserver service started")

			svc.Start()
		}
	})
}

// Stop will stop all internal services allowing the observer to exit cleanly.
func (o *PollingObserver) Close() error {
	o.stopOnce.Do(func() {
		o.cacheCleaner.Stop()
		for _, svc := range o.services {
			o.logger.Printf("PollingObserver service stopped")

			svc.Stop()
		}
	})

	return nil
}

func (o *PollingObserver) runHeadTasks() error {
	ch := o.heads.HeadTicker()
	for {
		select {
		case bl := <-ch:
			// limit the context timeout to configured value
			ctx, cancel := context.WithTimeout(o.ctx, o.samplingDuration)

			// run sampling with latest head
			o.processLatestHead(ctx, bl)

			// clean up resources by canceling the context after processing
			cancel()
		case <-o.ctx.Done():
			o.logger.Printf("PollingObserver.runHeadTasks ctx done")

			return o.ctx.Err()
		}
	}
}

// processLatestHead performs checking upkeep logic for all eligible keys of the given head
func (o *PollingObserver) processLatestHead(ctx context.Context, blockKey ocr2keepers.BlockKey) {
	var (
		keys []ocr2keepers.UpkeepKey
		ids  []ocr2keepers.UpkeepIdentifier
		err  error
	)
	o.logger.Printf("PollingObserver.processLatestHead")

	// Get only the active upkeeps from the id provider. This should not include
	// any cancelled upkeeps.
	if ids, err = o.src.GetActiveUpkeepIDs(ctx); err != nil {
		o.logger.Printf("%s: failed to get active upkeep ids from registry for sampling", err)
		return
	}

	o.logger.Printf("%d active upkeep ids found in registry", len(keys))

	keys = make([]ocr2keepers.UpkeepKey, len(ids))
	for i, id := range ids {
		keys[i] = o.enc.MakeUpkeepKey(blockKey, id)
	}

	// reduce keys to ratio size and shuffle. this can return a nil array.
	// in that case we have no keys so return.
	if keys = o.shuffleAndSliceKeysToRatio(keys); keys == nil {
		o.logger.Printf("PollingObserver.processLatestHead shuffleAndSliceKeysToRatio returned nil keys")

		return
	}

	o.stager.prepareBlock(blockKey)
	o.logger.Printf("PollingObserver.processLatestHead prepared block")

	// run checkupkeep on all keys. an error from this function should
	// bubble up.
	results, err := o.exe.CheckUpkeep(ctx, o.mercuryLookup, keys...)
	if err != nil {
		o.logger.Printf("%s: failed to parallel check upkeeps", err)
		return
	}

	for _, res := range results {
		key, _, err := o.enc.Detail(res)
		if err != nil {
			o.logger.Printf("error getting result detail: %s", err)
			continue
		}

		_, id, err := o.enc.SplitUpkeepKey(key)
		if err != nil {
			o.logger.Printf("error splitting upkeep key: %s", err)
		}

		o.stager.prepareIdentifier(id)
	}

	// advance the staged block/upkeep id list to the next in line
	o.stager.advance()
	o.logger.Printf("PollingObserver.processLatestHead advanced stager")
}

func (o *PollingObserver) shuffleAndSliceKeysToRatio(keys []ocr2keepers.UpkeepKey) []ocr2keepers.UpkeepKey {
	keys = o.shuffler.Shuffle(keys)
	size := o.ratio.OfInt(len(keys))

	if len(keys) == 0 || size <= 0 {
		o.logger.Printf("PollingObserver.shuffleAndSliceKeysToRatio returning nil")
		return nil
	}

	o.logger.Printf("PollingObserver.shuffleAndSliceKeysToRatio returning %d keys", len(keys[:size]))

	return keys[:size]
}

type stager struct {
	currentIDs   []ocr2keepers.UpkeepIdentifier
	currentBlock ocr2keepers.BlockKey
	nextIDs      []ocr2keepers.UpkeepIdentifier
	nextBlock    ocr2keepers.BlockKey
	sync.RWMutex
}

func (s *stager) prepareBlock(block ocr2keepers.BlockKey) {
	s.Lock()
	defer s.Unlock()

	s.nextBlock = block
}

func (s *stager) prepareIdentifier(id ocr2keepers.UpkeepIdentifier) {
	s.Lock()
	defer s.Unlock()

	if s.nextIDs == nil {
		s.nextIDs = []ocr2keepers.UpkeepIdentifier{}
	}

	s.nextIDs = append(s.nextIDs, id)
}

func (s *stager) advance() {
	s.Lock()
	defer s.Unlock()

	s.currentBlock = s.nextBlock
	s.currentIDs = make([]ocr2keepers.UpkeepIdentifier, len(s.nextIDs))

	copy(s.currentIDs, s.nextIDs)

	s.nextIDs = make([]ocr2keepers.UpkeepIdentifier, 0)
}

func (s *stager) get() (ocr2keepers.BlockKey, []ocr2keepers.UpkeepIdentifier) {
	s.RLock()
	defer s.RUnlock()

	return s.currentBlock, s.currentIDs
}
