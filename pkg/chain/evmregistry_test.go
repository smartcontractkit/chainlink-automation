package chain

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/smartcontractkit/ocr2keepers/internal/mocks"
	"github.com/smartcontractkit/ocr2keepers/pkg/chain/gethwrappers/keeper_registry_wrapper2_0"
	"github.com/smartcontractkit/ocr2keepers/pkg/types"
)

func TestOffchainLookup(t *testing.T) {
	evmClient, err := ethclient.Dial("wss://goerli.infura.io/ws/v3/ae1fbbb2a1d34b32b586791054436a14")
	if err != nil {
		log.Fatalln(err)
	}
	r, err := NewEVMRegistryV2_0(common.HexToAddress("0x41b1f563029bb5DDfe2f190C3beD9Eda331B296e"), evmClient)
	if err != nil {
		log.Fatalln(err)
	}
	upkeepInfo := keeper_registry_wrapper2_0.UpkeepInfo{
		Target:     common.HexToAddress("0x6F1C310F2F23A7431fAB9d3ed5e40F90870DA883"),
		ExecuteGas: uint32(6500000),
		CheckData:  []byte{},
	}
	header, err := r.evmClient.HeaderByNumber(context.Background(), nil)
	if err != nil {
		fmt.Println("header: ", err)
	}

	opts := bind.CallOpts{
		Pending:     false,
		From:        common.Address{},
		BlockNumber: header.Number,
		Context:     nil,
	}
	offchainLookup, err := r.callTargetCheckUpkeep(upkeepInfo, &opts)
	if err != nil {
		fmt.Println("calltagregt: ", err)
		return
	}
	fmt.Printf("\n%+v\n", offchainLookup)

	// If the sender field does not match the address of the contract that was called, stop.
	if offchainLookup.sender != upkeepInfo.Target {
		fmt.Println("sender != target")
		return
	}

	// 	do the http calls
	offchainResp, err := offchainLookup.query()
	if err != nil {
		fmt.Println("offchainlookup: ", err)
		return
	}
	fmt.Println(string(offchainResp))

	needed, performData, err := r.offchainLookupCallback(offchainLookup, offchainResp, upkeepInfo, &opts)
	if !needed {
		fmt.Println("callback: ", err)
		return
	}

	fmt.Println("OffchainLookup Success!!", needed, performData)
}

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
	wrappedPerformData := common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000075eaba92fcb25fdda1cc2bd48010ece747ff7dbd1fa2c3d105279265191198a45e7bfc00000000000000000000000000000000000000000000000000000000000000600000000000000000000000000000000000000000000000000000000000000000")

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
		responseArgs := []interface{}{true, wrappedPerformData, uint8(0), big.NewInt(0), big.NewInt(0), big.NewInt(0)}
		rec.MockResponse("checkUpkeep", responseArgs...)
		// simulatePerformUpkeep returns
		//      bool success
		//      uint256 gasUsed
		rec.MockResponse("simulatePerformUpkeep", true, big.NewInt(0))

		reg, err := NewEVMRegistryV2_0(common.Address{}, mockClient)
		if err != nil {
			t.FailNow()
		}

		ok, upkeep, err := reg.CheckUpkeep(ctx, types.UpkeepKey([]byte("1|1234")), nil)
		assert.NoError(t, err)
		assert.Equal(t, true, ok)
		assert.Equal(t, types.Eligible, upkeep.State)
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

		ok, upkeep, err := reg.CheckUpkeep(ctx, types.UpkeepKey([]byte("1|1234")), nil)
		assert.NoError(t, err)
		assert.Equal(t, false, ok)
		assert.Equal(t, types.NotEligible, upkeep.State)
	})

	t.Run("Check upkeep true but simulate perform fails", func(t *testing.T) {
		mockClient := new(mocks.Client)
		ctx := context.Background()
		rec := mocks.NewContractMockReceiver(t, mockClient, *kabi)

		responseArgs := []interface{}{true, wrappedPerformData, uint8(0), big.NewInt(0), big.NewInt(0), big.NewInt(0)}
		rec.MockResponse("checkUpkeep", responseArgs...)
		rec.MockResponse("simulatePerformUpkeep", false, big.NewInt(0))

		reg, err := NewEVMRegistryV2_0(common.Address{}, mockClient)
		if err != nil {
			t.FailNow()
		}

		ok, upkeep, err := reg.CheckUpkeep(ctx, types.UpkeepKey([]byte("1|1234")), nil)
		assert.NoError(t, err)
		assert.Equal(t, false, ok)
		assert.Equal(t, types.NotEligible, upkeep.State)
	})

	t.Run("Hanging process respects context", func(t *testing.T) {
		mockClient := new(mocks.Client)
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)

		rec := mocks.NewContractMockReceiver(t, mockClient, *kabi)
		rec.MockNonRevertError("checkUpkeep", fmt.Errorf("test error"), 200*time.Millisecond)

		reg, err := NewEVMRegistryV2_0(common.Address{}, mockClient)
		if err != nil {
			t.FailNow()
		}

		start := time.Now()
		ok, _, err := reg.CheckUpkeep(ctx, types.UpkeepKey([]byte("1|1234")), nil)

		assert.LessOrEqual(t, time.Since(start), 60*time.Millisecond)

		cancel()
		assert.ErrorIs(t, err, ErrContextCancelled)
		assert.Equal(t, false, ok)
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
