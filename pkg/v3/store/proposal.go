package store

import (
	"fmt"

	"github.com/smartcontractkit/ocr2keepers/pkg/util"
	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg/v3/types"
)

var (
	ErrMetadataUnavailable = fmt.Errorf("proposal recovery metadata unavailable")
	ErrInvalidValueType    = fmt.Errorf("invalid store value type")
)

type KeyValueGetter interface {
	Get(MetadataKey) (interface{}, bool)
}

func RecoveryProposalCacheFromMetadata(m KeyValueGetter) (*util.Cache[ocr2keepers.CoordinatedProposal], error) {
	rawArray, ok := m.Get(ProposalRecoveryMetadata)
	if !ok {
		return nil, ErrMetadataUnavailable
	}

	cache, ok := rawArray.(*util.Cache[ocr2keepers.CoordinatedProposal])
	if !ok {
		return nil, ErrInvalidValueType
	}

	return cache, nil
}

func SampleProposalsFromMetadata(m KeyValueGetter) ([]ocr2keepers.UpkeepIdentifier, error) {
	rawArray, ok := m.Get(ProposalSampleMetadata)
	if !ok {
		return nil, ErrMetadataUnavailable
	}

	ids, ok := rawArray.([]ocr2keepers.UpkeepIdentifier)
	if !ok {
		return nil, ErrInvalidValueType
	}

	return ids, nil
}
