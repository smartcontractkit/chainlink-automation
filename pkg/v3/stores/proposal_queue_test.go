package stores

import (
	"testing"

	"github.com/stretchr/testify/require"

	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg/v3/types"
)

func TestProposalQueue_Enqueue(t *testing.T) {
	tests := []struct {
		name      string
		initials  []ocr2keepers.CoordinatedProposal
		toEnqueue []ocr2keepers.CoordinatedProposal
		size      int
	}{
		{
			"add to empty queue",
			[]ocr2keepers.CoordinatedProposal{},
			[]ocr2keepers.CoordinatedProposal{
				{
					UpkeepID: upkeepId(ocr2keepers.LogTrigger, []byte{0x1}),
					WorkID:   "0x1",
				},
			},
			1,
		},
		{
			"add to non-empty queue",
			[]ocr2keepers.CoordinatedProposal{
				{
					UpkeepID: upkeepId(ocr2keepers.LogTrigger, []byte{0x1}),
					WorkID:   "0x1",
				},
			},
			[]ocr2keepers.CoordinatedProposal{
				{
					UpkeepID: upkeepId(ocr2keepers.LogTrigger, []byte{0x2}),
					WorkID:   "0x2",
				},
			},
			2,
		},
		{
			"add existing",
			[]ocr2keepers.CoordinatedProposal{
				{
					UpkeepID: upkeepId(ocr2keepers.LogTrigger, []byte{0x1}),
					WorkID:   "0x1",
				},
			},
			[]ocr2keepers.CoordinatedProposal{
				{
					UpkeepID: upkeepId(ocr2keepers.LogTrigger, []byte{0x1}),
					WorkID:   "0x1",
				},
			},
			1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			q := NewProposalQueue(func(uid ocr2keepers.UpkeepIdentifier) ocr2keepers.UpkeepType {
				return ocr2keepers.UpkeepType(uid[15])
			})

			require.NoError(t, q.Enqueue(tc.initials...))
			require.NoError(t, q.Enqueue(tc.toEnqueue...))
			require.Equal(t, tc.size, q.Size())
		})
	}
}

func TestProposalQueue_Dequeue(t *testing.T) {
	tests := []struct {
		name         string
		toEnqueue    []ocr2keepers.CoordinatedProposal
		dequeueType  ocr2keepers.UpkeepType
		dequeueCount int
		expected     []ocr2keepers.CoordinatedProposal
	}{
		{
			"empty queue",
			[]ocr2keepers.CoordinatedProposal{},
			ocr2keepers.LogTrigger,
			1,
			[]ocr2keepers.CoordinatedProposal{},
		},
		{
			"happy path log trigger",
			[]ocr2keepers.CoordinatedProposal{
				{
					UpkeepID: upkeepId(ocr2keepers.LogTrigger, []byte{0x1}),
					WorkID:   "0x1",
				},
				{
					UpkeepID: upkeepId(ocr2keepers.ConditionTrigger, []byte{0x1}),
					WorkID:   "0x2",
				},
			},
			ocr2keepers.LogTrigger,
			2,
			[]ocr2keepers.CoordinatedProposal{
				{
					UpkeepID: upkeepId(ocr2keepers.LogTrigger, []byte{0x1}),
					WorkID:   "0x1",
				},
			},
		},
		{
			"happy path log trigger",
			[]ocr2keepers.CoordinatedProposal{
				{
					UpkeepID: upkeepId(ocr2keepers.LogTrigger, []byte{0x1}),
					WorkID:   "0x1",
				},
				{
					UpkeepID: upkeepId(ocr2keepers.ConditionTrigger, []byte{0x1}),
					WorkID:   "0x2",
				},
			},
			ocr2keepers.ConditionTrigger,
			2,
			[]ocr2keepers.CoordinatedProposal{
				{
					UpkeepID: upkeepId(ocr2keepers.ConditionTrigger, []byte{0x1}),
					WorkID:   "0x2",
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			q := NewProposalQueue(func(uid ocr2keepers.UpkeepIdentifier) ocr2keepers.UpkeepType {
				return ocr2keepers.UpkeepType(uid[15])
			})
			for _, p := range tc.toEnqueue {
				q.Enqueue(p)
			}
			results, err := q.Dequeue(tc.dequeueType, tc.dequeueCount)
			require.NoError(t, err)
			require.Equal(t, len(tc.expected), len(results))

			for i := range tc.expected {
				require.Equal(t, tc.expected[i].WorkID, results[i].WorkID)
			}
		})
	}
}

func upkeepId(utype ocr2keepers.UpkeepType, rand []byte) ocr2keepers.UpkeepIdentifier {
	id := [32]byte{}
	id[15] = byte(utype)
	copy(id[16:], rand)
	return ocr2keepers.UpkeepIdentifier(id)
}
