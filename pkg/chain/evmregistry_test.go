package chain

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/ocr2keepers/pkg/chain/gethwrappers/keeper_registry_wrapper2_0"
	"github.com/smartcontractkit/ocr2keepers/pkg/types"
)

func TestGetActiveUpkeepKeys(t *testing.T) {
	mockClient := types.NewMockEVMClient(t)
	ctx := context.Background()
	kabi, _ := keeper_registry_wrapper2_0.KeeperRegistryMetaData.GetAbi()
	rec := NewContractMockReceiver(t, mockClient, *kabi)

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

	b := BlockKey("0")

	blockKey, keys, err := reg.GetActiveUpkeepIDs(ctx, b)
	if err != nil {
		t.Logf("error: %s", err)
		t.FailNow()
	}

	assert.Len(t, keys, 4)
	mockClient.Mock.AssertExpectations(t)
}

func TestCheckUpkeep(t *testing.T) {
	wrappedPerformData := common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000006000000000000000000000000000000000000000000000000000000000000000a00000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000075eaba92fcb25fdda1cc2bd48010ece747ff7dbd1fa2c3d105279265191198a45e7bfc00000000000000000000000000000000000000000000000000000000000000600000000000000000000000000000000000000000000000000000000000000000")

	var ret0 = new(res)
	err := pdataABI.UnpackIntoInterface(ret0, "check", wrappedPerformData)
	require.NoError(t, err)

	upkeepKey := UpkeepKey("1|1234")
	_, expectedUpkeep, err := upkeepKey.BlockKeyAndUpkeepID()
	require.NoError(t, err)

	upkeepId, ok := expectedUpkeep.BigInt()
	require.True(t, ok)

	checkPayload, err := keeperRegistryABI.Pack("checkUpkeep", upkeepId)
	require.NoError(t, err)

	performPayload, err := keeperRegistryABI.Pack("simulatePerformUpkeep", upkeepId, ret0.Result.PerformData)
	require.NoError(t, err)

	t.Run("Perform", func(t *testing.T) {
		mockClient := types.NewMockEVMClient(t)
		ctx := context.Background()

		reg, err := NewEVMRegistryV2_0(common.Address{}, mockClient)
		require.NoError(t, err)

		// checkUpkeep returns
		//      bool upkeepNeeded,
		//      bytes memory performData,
		//      uint8 upkeepFailureReason,
		//      uint256 gasUsed,
		//      uint256 fastGasWei,
		//      uint256 linkNative
		responseArgs := []interface{}{true, wrappedPerformData, uint8(0), big.NewInt(0), big.NewInt(0), big.NewInt(0)}
		mockClient.On("BatchCallContext", ctx, mock.Anything).
			Once().
			Run(func(args mock.Arguments) {
				batchElems, ok := args.Get(1).([]rpc.BatchElem)
				assert.True(t, ok)
				assert.Len(t, batchElems, 1)
				for _, batchElem := range batchElems {
					assert.Equal(t, "eth_call", batchElem.Method)
					assert.IsType(t, new(string), batchElem.Result)
					assert.Len(t, batchElem.Args, 2)
					assert.Equal(t, map[string]interface{}{
						"to":   common.Address{}.Hex(),
						"data": hexutil.Bytes(checkPayload),
					}, batchElem.Args[0])
					assert.Equal(t, hexutil.EncodeBig(big.NewInt(1)), batchElem.Args[1])

					out, err := keeperRegistryABI.Methods["checkUpkeep"].
						Outputs.PackValues(responseArgs)
					assert.NoError(t, err)

					*batchElem.Result.(*string) = hexutil.Encode(out)
				}
			}).Return(nil)

		// simulatePerformUpkeep returns
		//      bool success
		//      uint256 gasUsed
		mockClient.On("BatchCallContext", ctx, mock.Anything).
			Once().
			Run(func(args mock.Arguments) {
				batchElems, ok := args.Get(1).([]rpc.BatchElem)
				assert.True(t, ok)
				assert.Len(t, batchElems, 1)
				for _, batchElem := range batchElems {
					assert.Equal(t, "eth_call", batchElem.Method)
					assert.IsType(t, new(string), batchElem.Result)
					assert.Len(t, batchElem.Args, 2)
					assert.Equal(t, map[string]interface{}{
						"to":   common.Address{}.Hex(),
						"data": hexutil.Bytes(performPayload),
					}, batchElem.Args[0])
					assert.Equal(t, hexutil.EncodeBig(big.NewInt(1)), batchElem.Args[1])

					out, err := keeperRegistryABI.Methods["simulatePerformUpkeep"].
						Outputs.PackValues([]interface{}{true, big.NewInt(0)})
					assert.NoError(t, err)

					*batchElem.Result.(*string) = hexutil.Encode(out)
				}
			}).Return(nil)

		upkeep, err := reg.CheckUpkeep(ctx, upkeepKey)
		assert.NoError(t, err)
		assert.Len(t, upkeep, 1)
		assert.Equal(t, types.Eligible, upkeep[0].State)
	})

	t.Run("UPKEEP_NOT_NEEDED", func(t *testing.T) {
		mockClient := types.NewMockEVMClient(t)
		ctx := context.Background()

		reg, err := NewEVMRegistryV2_0(common.Address{}, mockClient)
		require.NoError(t, err)

		// checkUpkeep returns
		//      bool upkeepNeeded,
		//      bytes memory performData,
		//      uint8 upkeepFailureReason,
		//      uint256 gasUsed,
		//      uint256 fastGasWei,
		//      uint256 linkNative
		responseArgs := []interface{}{false, []byte{}, uint8(4), big.NewInt(0), big.NewInt(0), big.NewInt(0)}
		mockClient.On("BatchCallContext", ctx, mock.Anything).
			Once().
			Run(func(args mock.Arguments) {
				batchElems, ok := args.Get(1).([]rpc.BatchElem)
				assert.True(t, ok)
				assert.Len(t, batchElems, 1)
				for _, batchElem := range batchElems {
					assert.Equal(t, "eth_call", batchElem.Method)
					assert.IsType(t, new(string), batchElem.Result)
					assert.Len(t, batchElem.Args, 2)
					assert.Equal(t, map[string]interface{}{
						"to":   common.Address{}.Hex(),
						"data": hexutil.Bytes(checkPayload),
					}, batchElem.Args[0])
					assert.Equal(t, hexutil.EncodeBig(big.NewInt(1)), batchElem.Args[1])

					out, err := keeperRegistryABI.Methods["checkUpkeep"].
						Outputs.PackValues(responseArgs)
					assert.NoError(t, err)

					*batchElem.Result.(*string) = hexutil.Encode(out)
				}
			}).Return(nil)

		upkeep, err := reg.CheckUpkeep(ctx, upkeepKey)
		assert.NoError(t, err)
		assert.Len(t, upkeep, 1)
		assert.Equal(t, types.NotEligible, upkeep[0].State)

		mockClient.AssertExpectations(t)
	})

	t.Run("Check upkeep true but simulate perform fails", func(t *testing.T) {
		mockClient := types.NewMockEVMClient(t)
		ctx := context.Background()

		reg, err := NewEVMRegistryV2_0(common.Address{}, mockClient)
		require.NoError(t, err)

		// checkUpkeep returns
		//      bool upkeepNeeded,
		//      bytes memory performData,
		//      uint8 upkeepFailureReason,
		//      uint256 gasUsed,
		//      uint256 fastGasWei,
		//      uint256 linkNative
		responseArgs := []interface{}{true, wrappedPerformData, uint8(0), big.NewInt(0), big.NewInt(0), big.NewInt(0)}
		mockClient.On("BatchCallContext", ctx, mock.Anything).
			Once().
			Run(func(args mock.Arguments) {
				batchElems, ok := args.Get(1).([]rpc.BatchElem)
				assert.True(t, ok)
				assert.Len(t, batchElems, 1)
				for _, batchElem := range batchElems {
					assert.Equal(t, "eth_call", batchElem.Method)
					assert.IsType(t, new(string), batchElem.Result)
					assert.Len(t, batchElem.Args, 2)
					assert.Equal(t, map[string]interface{}{
						"to":   common.Address{}.Hex(),
						"data": hexutil.Bytes(checkPayload),
					}, batchElem.Args[0])
					assert.Equal(t, hexutil.EncodeBig(big.NewInt(1)), batchElem.Args[1])

					out, err := keeperRegistryABI.Methods["checkUpkeep"].
						Outputs.PackValues(responseArgs)
					assert.NoError(t, err)

					*batchElem.Result.(*string) = hexutil.Encode(out)
				}
			}).Return(nil)

		// simulatePerformUpkeep returns
		//      bool success
		//      uint256 gasUsed
		mockClient.On("BatchCallContext", ctx, mock.Anything).
			Once().
			Run(func(args mock.Arguments) {
				batchElems, ok := args.Get(1).([]rpc.BatchElem)
				assert.True(t, ok)
				assert.Len(t, batchElems, 1)
				for _, batchElem := range batchElems {
					assert.Equal(t, "eth_call", batchElem.Method)
					assert.IsType(t, new(string), batchElem.Result)
					assert.Len(t, batchElem.Args, 2)
					assert.Equal(t, map[string]interface{}{
						"to":   common.Address{}.Hex(),
						"data": hexutil.Bytes(performPayload),
					}, batchElem.Args[0])
					assert.Equal(t, hexutil.EncodeBig(big.NewInt(1)), batchElem.Args[1])

					out, err := keeperRegistryABI.Methods["simulatePerformUpkeep"].
						Outputs.PackValues([]interface{}{false, big.NewInt(0)})
					assert.NoError(t, err)

					*batchElem.Result.(*string) = hexutil.Encode(out)
				}
			}).Return(nil)

		upkeep, err := reg.CheckUpkeep(ctx, upkeepKey)
		assert.NoError(t, err)
		assert.Len(t, upkeep, 1)
		assert.Equal(t, types.NotEligible, upkeep[0].State)
	})

	t.Run("Hanging process respects context", func(t *testing.T) {
		mockClient := types.NewMockEVMClient(t)
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)

		mockClient.On("BatchCallContext", ctx, mock.Anything).
			Once().
			Run(func(args mock.Arguments) {
				time.Sleep(200 * time.Millisecond)
			}).
			Return(nil)

		reg, err := NewEVMRegistryV2_0(common.Address{}, mockClient)
		require.NoError(t, err)

		start := time.Now()
		_, err = reg.CheckUpkeep(ctx, upkeepKey)

		assert.LessOrEqual(t, time.Since(start), 60*time.Millisecond)

		cancel()
		assert.ErrorIs(t, err, ErrContextCancelled)
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
