package ocr2keepers

// ResultStore stores check results and allows for querying and removing result
type ResultStore interface {
	Add(...CheckResult)
	Remove(...string)
	View(...Filter) ([]CheckResult, error)
}

// Filter is a function that filters check results from a ResultStore
type Filter func(CheckResult) bool

// NotifyOp is an operation that can be notified by the ResultStore
type NotifyOp uint8

const (
	NotifyOpNil NotifyOp = iota
	// NotifyOpEvict is a notification that a result has been evicted from the store after TTL has passed
	NotifyOpEvict
	// NotifyOpRemove is a notification that a result has been removed from the store
	NotifyOpRemove
)

// Notification is a struct that will be sent by the ResultStore upon certain events happening
type Notification struct {
	Op   NotifyOp
	Data CheckResult
}
