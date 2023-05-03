package logs

import (
	"math/big"
	"math/rand"
	"testing"

	"github.com/smartcontractkit/ocr2keepers/pkg/chain"
	"github.com/smartcontractkit/ocr2keepers/pkg/types"
	"github.com/stretchr/testify/require"
)

func TestUpkeepsQueue_e2e(t *testing.T) {
	q := NewUpkeepsQueue()

	block := big.NewInt(1)
	q.Push(randUpkeepResult(block), randUpkeepResult(block))
	require.Equal(t, 2, q.Size())
	require.Equal(t, 0, q.SizeVisited())

	block = block.Add(block, big.NewInt(1))
	q.Push(randUpkeepResult(block), randUpkeepResult(block))
	require.Equal(t, 4, q.Size())
	require.Equal(t, 0, q.SizeVisited())

	results := q.Pop(2)
	require.Len(t, results, 2)
	require.Equal(t, 2, q.Size())
	require.Equal(t, 2, q.SizeVisited())

	results = q.PopVisited(2)
	require.Len(t, results, 2)
	require.Equal(t, 2, q.Size())
	require.Equal(t, 0, q.SizeVisited())
}

func randUpkeepResult(block *big.Int) types.UpkeepResult {
	return types.UpkeepResult{
		Key: chain.NewUpkeepKey(block, big.NewInt(rand.Int63n(1e9))),
	}
}
