package ocr2keepers

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
	Add(...CheckResult)
	Remove(...string)
	View(...ViewOpt) ([]CheckResult, error)
	Notifications() <-chan Notification
}

// Notification is a struct that will be sent by the ResultStore upon certain events happening
type Notification struct {
	Op   NotifyOp
	Data CheckResult
}

// Filter is a function that filters check results from a ResultStore
type ResultFilter func(CheckResult) bool

// Comparator is a function that is used for ordering of results from a ResultStore.
// It should return true if the first result should be ordered before the second result.
type ResultComparator func(i, j CheckResult) bool

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
