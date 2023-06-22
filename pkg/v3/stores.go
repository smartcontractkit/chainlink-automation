package ocr2keepers

import (
	"sync"

	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg"
	"github.com/smartcontractkit/ocr2keepers/pkg/v3/tickers"
)

// NotifyOp is an operation that can be notified by the ResultStore
type NotifyOp uint8

const (
	NotifyOpNil NotifyOp = iota
	// NotifyOpEvict is a notification that a result has been evicted from the store after TTL has passed
	NotifyOpEvict
	// NotifyOpRemove is a notification that a result has been removed from the store
	NotifyOpRemove
)

// ResultStore stores check results and allows for querying and removing result
type ResultStore interface {
	Add(...ocr2keepers.CheckResult)
	Remove(...string)
	View(...ViewOpt) ([]ocr2keepers.CheckResult, error)
	Notifications() <-chan Notification
}

type InstructionStore interface{}

type MetadataStore interface {
	// Set should replace any existing values
	Set([]ocr2keepers.UpkeepIdentifier) error
	// Start should begin watching for new block history
	Start() error
	// Stop should stop watching for new block heights
	Stop() error
}

// Notification is a struct that will be sent by the ResultStore upon certain events happening
type Notification struct {
	Op   NotifyOp
	Data ocr2keepers.CheckResult
}

// Filter is a function that filters check results from a ResultStore
type ResultFilter func(ocr2keepers.CheckResult) bool

// Comparator is a function that is used for ordering of results from a ResultStore.
// It should return true if the first result should be ordered before the second result.
type ResultComparator func(i, j ocr2keepers.CheckResult) bool

// ViewOpts is a set of options that can be passed to the View method of a ResultStore
type viewOpts struct {
	filters     []ResultFilter
	comparators []ResultComparator
	limit       int
}

type ViewOpts []ViewOpt

// Apply applies the ViewOpts to a viewOpts and destructs it into filters, comparators and limit.
// TODO: TBD if we want to keep this or just use the viewOpts directly or as arguments in the View method.
func (vo ViewOpts) Apply() ([]ResultFilter, []ResultComparator, int) {
	opts := &viewOpts{}
	for _, opt := range vo {
		opt(opts)
	}
	return opts.filters, opts.comparators, opts.limit
}

// ViewOpt is an option that can be passed to the View method of a ResultStore
type ViewOpt func(*viewOpts)

// WithResultFilters enables to specify filters that will be applied to the results
func WithFilter(filters ...ResultFilter) ViewOpt {
	return func(opts *viewOpts) {
		opts.filters = append(opts.filters, filters...)
	}
}

// WithOrder enables to specify the order in which the results should be returned
func WithOrder(comparators ...ResultComparator) ViewOpt {
	return func(opts *viewOpts) {
		opts.comparators = append(opts.comparators, comparators...)
	}
}

// WithOrderView enables to limit the amount of results that should be returned
func WithLimit(limit int) ViewOpt {
	return func(opts *viewOpts) {
		opts.limit = limit
	}
}

type metadataStore struct {
	ticker       *tickers.BlockTicker
	identifiers  []ocr2keepers.UpkeepIdentifier
	blockHistory ocr2keepers.BlockHistory
	stopCh       chan int
	m            sync.RWMutex
}

func NewMetadataStore(ticker *tickers.BlockTicker) *metadataStore {
	stopCh := make(chan int)
	return &metadataStore{
		ticker: ticker,
		stopCh: stopCh,
	}
}

func (s *metadataStore) Set(identifiers []ocr2keepers.UpkeepIdentifier) error {
	s.m.Lock()
	defer s.m.Unlock()

	s.identifiers = identifiers
	return nil
}

func (s *metadataStore) setBlockHistory(history ocr2keepers.BlockHistory) {
	s.m.Lock()
	defer s.m.Unlock()

	s.blockHistory = history
}

func (s *metadataStore) Start() error {
loop:
	for {
		select {
		case blockHistory := <-s.ticker.C:
			s.setBlockHistory(blockHistory)
		case <-s.stopCh:
			break loop
		}
	}
	return nil
}

func (s *metadataStore) Stop() error {
	s.stopCh <- 1
	return nil
}
