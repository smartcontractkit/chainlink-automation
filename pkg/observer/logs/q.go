package logs

import (
	"github.com/smartcontractkit/ocr2keepers/internal/util"
	"github.com/smartcontractkit/ocr2keepers/pkg/types"
)

// LogUpkeepsQueue holds internal queues to manage currently executed upkeeps
type LogUpkeepsQueue struct {
	base    *util.Queue[types.UpkeepResult]
	visited *util.Queue[types.UpkeepResult]
}

func NewUpkeepsQueue() *LogUpkeepsQueue {
	return &LogUpkeepsQueue{
		base:    util.NewQueue[types.UpkeepResult](),
		visited: util.NewQueue[types.UpkeepResult](),
	}
}

// Push adds items to the q, it is possible to add values of multiple buckets
func (uq *LogUpkeepsQueue) Push(vals ...types.UpkeepResult) {
	uq.base.Push(vals...)
}

// Pop returns the corresponding items and removed them from the q
func (uq *LogUpkeepsQueue) Pop(n int) []types.UpkeepResult {
	removed := uq.base.Pop(n)
	uq.visited.Push(removed...)
	return removed
}

func (uq *LogUpkeepsQueue) Size() int {
	return uq.base.Size()
}

// Visited returns the results that were already visited in the past
func (uq *LogUpkeepsQueue) Visited() *util.Queue[types.UpkeepResult] {
	return uq.visited
}
