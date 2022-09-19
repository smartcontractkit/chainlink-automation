package chain

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/smartcontractkit/ocr2keepers/internal/mocks"
	"github.com/smartcontractkit/ocr2keepers/pkg/chain/gethwrappers/keeper_registry_wrapper2_0"
	"github.com/smartcontractkit/ocr2keepers/pkg/types"
)

func TestGetActiveUpkeepKeys(t *testing.T) {
	mockClient := new(mocks.Client)
	ctx := context.Background()
	kabi, _ := keeper_registry_wrapper2_0.KeeperRegistryMetaData.GetAbi()
	rec := mocks.NewContractMockReceiver(t, mockClient, *kabi)

	block := big.NewInt(4)
	mockClient.On("HeaderByNumber", ctx, mock.Anything).Return(&ethtypes.Header{Number: block}, nil).Once()

	state := MockGetState
	state.State.NumUpkeeps = big.NewInt(4)
	ids := []*big.Int{big.NewInt(1), big.NewInt(2), big.NewInt(3), big.NewInt(4)}

	rec.MockResponse("getState", state)
	rec.MockResponse("getActiveUpkeepIDs", ids)

	reg, err := NewEVMRegistryV2_0(common.Address{}, mockClient)
	if err != nil {
		t.FailNow()
	}

	keys, err := reg.GetActiveUpkeepKeys(ctx, types.BlockKey("0"))
	if err != nil {
		t.Logf("error: %s", err)
		t.FailNow()
	}

	assert.Len(t, keys, 4)
	mockClient.Mock.AssertExpectations(t)
}

func TestCheckUpkeep(t *testing.T) {
	kabi, _ := keeper_registry_wrapper2_0.KeeperRegistryMetaData.GetAbi()

	t.Run("Perform", func(t *testing.T) {
		mockClient := new(mocks.Client)
		ctx := context.Background()
		rec := mocks.NewContractMockReceiver(t, mockClient, *kabi)

		// checkUpkeep returns
		//      bool upkeepNeeded,
		//      bytes memory performData,
		//      uint8 upkeepFailureReason,
		//      uint256 gasUsed,
		//      uint256 fastGasWei,
		//      uint256 linkNative
		responseArgs := []interface{}{true, []byte{}, uint8(0), big.NewInt(0), big.NewInt(0), big.NewInt(0)}
		rec.MockResponse("checkUpkeep", responseArgs...)

		reg, err := NewEVMRegistryV2_0(common.Address{}, mockClient)
		if err != nil {
			t.FailNow()
		}

		ok, upkeep, err := reg.CheckUpkeep(ctx, types.UpkeepKey([]byte("1|1234")))
		assert.NoError(t, err)
		assert.Equal(t, true, ok)
		assert.Equal(t, types.Perform, upkeep.State)
	})

	t.Run("UPKEEP_NOT_NEEDED", func(t *testing.T) {
		mockClient := new(mocks.Client)
		ctx := context.Background()
		rec := mocks.NewContractMockReceiver(t, mockClient, *kabi)

		// checkUpkeep returns
		//      bool upkeepNeeded,
		//      bytes memory performData,
		//      uint8 upkeepFailureReason,
		//      uint256 gasUsed,
		//      uint256 fastGasWei,
		//      uint256 linkNative
		responseArgs := []interface{}{false, []byte{}, uint8(4), big.NewInt(0), big.NewInt(0), big.NewInt(0)}
		rec.MockResponse("checkUpkeep", responseArgs...)

		reg, err := NewEVMRegistryV2_0(common.Address{}, mockClient)
		if err != nil {
			t.FailNow()
		}

		ok, upkeep, err := reg.CheckUpkeep(ctx, types.UpkeepKey([]byte("1|1234")))
		assert.NoError(t, err)
		assert.Equal(t, false, ok)
		assert.Equal(t, types.Skip, upkeep.State)
	})

}

var MockRegistryState = keeper_registry_wrapper2_0.State{
	Nonce:                   uint32(0),
	OwnerLinkBalance:        big.NewInt(1000000000000000000),
	ExpectedLinkBalance:     big.NewInt(1000000000000000000),
	NumUpkeeps:              big.NewInt(0),
	TotalPremium:            big.NewInt(100),
	ConfigCount:             uint32(0),
	LatestConfigBlockNumber: uint32(0),
	LatestConfigDigest:      [32]byte{},
	LatestEpoch:             0,
	Paused:                  false,
}

var MockRegistryConfig = keeper_registry_wrapper2_0.OnchainConfig{
	PaymentPremiumPPB:    100,
	FlatFeeMicroLink:     uint32(0),
	CheckGasLimit:        2_000_000,
	StalenessSeconds:     big.NewInt(3600),
	GasCeilingMultiplier: uint16(2),
	MinUpkeepSpend:       big.NewInt(0),
	MaxPerformGas:        uint32(5000000),
	MaxCheckDataSize:     uint32(5000),
	MaxPerformDataSize:   uint32(5000),
	FallbackGasPrice:     big.NewInt(1000000),
	FallbackLinkPrice:    big.NewInt(1000000),
	Transcoder:           common.Address{},
	Registrar:            common.Address{},
}

var MockGetState = keeper_registry_wrapper2_0.GetState{
	State:        MockRegistryState,
	Config:       MockRegistryConfig,
	Signers:      []common.Address{},
	Transmitters: []common.Address{},
	F:            uint8(4),
}
