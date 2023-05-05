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
	u1, u2 := randUpkeepResult(block), randUpkeepResult(block)
	q.Push(u1, u2)
	require.Equal(t, 2, q.Size())
	require.Equal(t, 0, q.Visited())

	results := q.Pop(2)
	require.Len(t, results, 2)
	require.Equal(t, 0, q.Size())
	require.Equal(t, 2, q.Visited())

	// cleaning one upkeep, the other one goes back to q
	keysMap := make(map[string]bool)
	keysMap[u1.Key.String()] = true
	q.Clean(func(ur types.UpkeepResult) bool {
		return keysMap[ur.Key.String()]
	})
	require.Equal(t, 1, q.Size())
	require.Equal(t, 0, q.Visited())
}

func randUpkeepResult(block *big.Int) types.UpkeepResult {
	return types.UpkeepResult{
		Key: chain.NewUpkeepKey(block, big.NewInt(rand.Int63n(1e9))),
	}
}
