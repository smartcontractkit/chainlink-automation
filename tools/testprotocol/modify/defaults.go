package modify

import types "github.com/smartcontractkit/chainlink-common/pkg/types/automation"

var (
	ObservationModifiers = []NamedModifier{
		Modify(
			"set all proposals to block 1",
			AsObservation(
				WithProposalBlockAs(types.BlockNumber(1)))),
		Modify(
			"set all proposals to block 1_000_000_000",
			AsObservation(
				WithProposalBlockAs(types.BlockNumber(1_000_000_000)))),
		Modify(
			"set all proposals to block 0",
			AsObservation(
				WithProposalBlockAs(types.BlockNumber(0)))),
		Modify(
			"set all performables to block 1",
			AsObservation(
				WithPerformableBlockAs(types.BlockNumber(1)))),
		Modify(
			"set all performables to block 1_000_000_000",
			AsObservation(
				WithPerformableBlockAs(types.BlockNumber(1_000_000_000)))),
		Modify(
			"set all performables to block 0",
			AsObservation(
				WithPerformableBlockAs(types.BlockNumber(0)))),
		Modify(
			"set all block history numbers to 0",
			AsObservation(
				WithBlockHistoryBlockAs(0))),
		Modify(
			"set all block history numbers to 1",
			AsObservation(
				WithBlockHistoryBlockAs(1))),
		Modify(
			"set all block history numbers to 1_000_000_000",
			AsObservation(
				WithBlockHistoryBlockAs(1_000_000_000))),
	}

	OutcomeModifiers = []NamedModifier{
		Modify(
			"set all proposals to block 1",
			AsOutcome(
				WithProposalBlockAs(types.BlockNumber(1)))),
		Modify(
			"set all proposals to block 1_000_000_000",
			AsOutcome(
				WithProposalBlockAs(types.BlockNumber(1_000_000_000)))),
		Modify(
			"set all proposals to block 0",
			AsOutcome(
				WithProposalBlockAs(types.BlockNumber(0)))),
		Modify(
			"set all performables to block 1",
			AsOutcome(
				WithPerformableBlockAs(types.BlockNumber(1)))),
		Modify(
			"set all performables to block 1_000_000_000",
			AsOutcome(
				WithPerformableBlockAs(types.BlockNumber(1_000_000_000)))),
		Modify(
			"set all performables to block 0",
			AsOutcome(
				WithPerformableBlockAs(types.BlockNumber(0)))),
	}

	ObservationInvalidValueModifiers = []NamedByteModifier{
		ModifyBytes(
			"set block value to empty string",
			WithModifyKeyValue("BlockNumber", func(_ string, value interface{}) interface{} {
				return ""
			})),
		ModifyBytes(
			"set block value to negative number",
			WithModifyKeyValue("BlockNumber", func(_ string, value interface{}) interface{} {
				return -1
			})),
		ModifyBytes(
			"set block value to very large number as string",
			WithModifyKeyValue("BlockNumber", func(_ string, value interface{}) interface{} {
				return "98989898989898989898989898989898989898989898"
			})),
	}

	InvalidBlockModifiers = []NamedByteModifier{
		ModifyBytes(
			"set block value to empty string",
			WithModifyKeyValue("BlockNumber", func(_ string, value interface{}) interface{} {
				return ""
			})),
		ModifyBytes(
			"set block value to negative number",
			WithModifyKeyValue("BlockNumber", func(_ string, value interface{}) interface{} {
				return -1
			})),
		ModifyBytes(
			"set block value to very large number as string",
			WithModifyKeyValue("BlockNumber", func(_ string, value interface{}) interface{} {
				return "98989898989898989898989898989898989898989898"
			})),
	}
)
