package chain

import (
	"bytes"
	"math/big"
	"math/rand"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	ktypes "github.com/smartcontractkit/ocr2keepers/pkg/types"
	"github.com/stretchr/testify/assert"
)

func TestNewEVMEncoder(t *testing.T) {
	enc := NewEVMReportEncoder()
	assert.NotNil(t, enc)
}

func TestEncodeReport_MultiplePerforms(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		input := []ktypes.UpkeepResult{
			{
				Key:              ktypes.UpkeepKey("42|18"),
				PerformData:      []byte("hello"),
				FastGasWei:       big.NewInt(16),
				LinkNative:       big.NewInt(8),
				CheckBlockNumber: 42,
				CheckBlockHash:   [32]byte{1},
			},
			{
				Key:              ktypes.UpkeepKey("43|23"),
				PerformData:      []byte("long perform data that takes up more than 32 bytes to show how byte arrays are abi encoded. this should take up multiple slots."),
				FastGasWei:       big.NewInt(8),
				LinkNative:       big.NewInt(16),
				CheckBlockNumber: 43,
				CheckBlockHash:   [32]byte{2},
			},
		}

		encoder := &evmReportEncoder{}
		b, err := encoder.EncodeReport(input)

		// fast gas and link native values should come from the result at the latest block number
		buff := new(bytes.Buffer)
		// buff.Write([]byte("0x"))                                                                         // preface to data structure
		buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000008")) // fastGasWei in hex format
		buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000010")) // linkNative in hex format
		buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000080")) // offset for ids array
		buff.Write(common.Hex2Bytes("00000000000000000000000000000000000000000000000000000000000000e0")) // offset for tuple array
		buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000002")) // array length of 2
		buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000012")) // first upkeep id - 18
		buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000017")) // second upkeep id - 23
		buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000002")) // length of array 2
		buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000040")) // tuple 1 offset
		buff.Write(common.Hex2Bytes("00000000000000000000000000000000000000000000000000000000000000e0")) // tuple 2 offset
		buff.Write(common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000000002a")) // block number 42
		buff.Write(common.Hex2Bytes("0100000000000000000000000000000000000000000000000000000000000000")) // block hash 1
		buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000060")) // tuple 1 byte array offset
		buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000005")) // tuple 1 byte length
		buff.Write(common.Hex2Bytes("68656c6c6f000000000000000000000000000000000000000000000000000000")) // tuple 1 bytes
		buff.Write(common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000000002b")) // block number 43
		buff.Write(common.Hex2Bytes("0200000000000000000000000000000000000000000000000000000000000000")) // block hash 2
		buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000060")) // tuple 2 byte array offset
		buff.Write(common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000000007f")) // tuple 2 byte length
		buff.Write(common.Hex2Bytes("6c6f6e6720706572666f726d206461746120746861742074616b657320757020")) // tuple 2 bytes
		buff.Write(common.Hex2Bytes("6d6f7265207468616e20333220627974657320746f2073686f7720686f772062")) // tuple 2 bytes cont...
		buff.Write(common.Hex2Bytes("79746520617272617973206172652061626920656e636f6465642e2074686973")) // tuple 2 bytes cont...
		buff.Write(common.Hex2Bytes("2073686f756c642074616b65207570206d756c7469706c6520736c6f74732e00")) // tuple 2 bytes cont...

		expected := buff.Bytes()

		assert.NoError(t, err)
		assert.Equal(t, expected, b)
	})

	t.Run("Key Parse Error", func(t *testing.T) {
		input := []ktypes.UpkeepResult{
			{Key: ktypes.UpkeepKey([]byte("1")), PerformData: []byte("hello")},
		}

		encoder := &evmReportEncoder{}
		b, err := encoder.EncodeReport(input)

		assert.ErrorIs(t, err, ErrUpkeepKeyNotParsable)
		assert.Equal(t, []byte(nil), b)
	})
}

func TestEncodeReport_EmptyPerformData(t *testing.T) {
	input := []ktypes.UpkeepResult{
		{
			Key:              ktypes.UpkeepKey([]byte("43|18")),
			PerformData:      []byte{},
			FastGasWei:       big.NewInt(8),
			LinkNative:       big.NewInt(16),
			CheckBlockNumber: 43,
			CheckBlockHash:   [32]byte{2},
		},
	}

	encoder := &evmReportEncoder{}
	b, err := encoder.EncodeReport(input)

	// fast gas and link native values should come from the result at the latest block number
	buff := new(bytes.Buffer)
	buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000008")) // fastGast
	buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000010")) // link native
	buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000080")) // ids array offset
	buff.Write(common.Hex2Bytes("00000000000000000000000000000000000000000000000000000000000000c0")) // tuple array offset
	buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000001")) // id array length
	buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000012")) // id
	buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000001")) // tuple array length
	buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000020")) // tuple 1 offset
	buff.Write(common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000000002b")) // block number 43
	buff.Write(common.Hex2Bytes("0200000000000000000000000000000000000000000000000000000000000000")) // block hash 2
	buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000060")) // tuple bytes offset
	buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000000")) // tuple bytes

	expected := buff.Bytes()

	assert.NoError(t, err)
	assert.Equal(t, expected, b)
}

func TestDecodeReport(t *testing.T) {
	expected := []ktypes.UpkeepResult{
		{
			Key:              ktypes.UpkeepKey([]byte("43|18")),
			State:            ktypes.Eligible, // it is assumed that all items in a report were eligible
			PerformData:      []byte{},
			FastGasWei:       big.NewInt(8),
			LinkNative:       big.NewInt(16),
			CheckBlockNumber: 43,
			CheckBlockHash:   [32]byte{2},
		},
	}

	enc := &evmReportEncoder{}

	// fast gas and link native values should come from the result at the latest block number
	buff := new(bytes.Buffer)
	buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000008")) // fastGast
	buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000010")) // link native
	buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000080")) // ids array offset
	buff.Write(common.Hex2Bytes("00000000000000000000000000000000000000000000000000000000000000c0")) // tuple array offset
	buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000001")) // id array length
	buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000012")) // id
	buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000001")) // tuple array length
	buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000020")) // tuple 1 offset
	buff.Write(common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000000002b")) // block number 43
	buff.Write(common.Hex2Bytes("0200000000000000000000000000000000000000000000000000000000000000")) // block hash 2
	buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000060")) // tuple bytes offset
	buff.Write(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000000")) // tuple bytes

	ids, err := enc.DecodeReport(buff.Bytes())

	assert.NoError(t, err)
	assert.Equal(t, expected, ids)
}

func BenchmarkEncodeReport(b *testing.B) {
	key1 := ktypes.UpkeepKey([]byte("1239428374|187689279234987"))
	key2 := ktypes.UpkeepKey([]byte("1239428374|187689279234989"))
	key3 := ktypes.UpkeepKey([]byte("1239428375|187689279234987"))

	noData := []byte{}
	smallData := make([]byte, 12)
	largeData := make([]byte, 128)
	zeroBigInt := big.NewInt(0)

	rand.Read(smallData)
	rand.Read(largeData)

	encoder := NewEVMReportEncoder()
	tests := []struct {
		Name string
		Data []ktypes.UpkeepResult
	}{
		{Name: "No Perform Data", Data: []ktypes.UpkeepResult{
			{Key: key1, PerformData: noData, GasUsed: zeroBigInt, FastGasWei: zeroBigInt, LinkNative: zeroBigInt, CheckBlockNumber: 1239428374},
		}},
		{Name: "Small Perform Data", Data: []ktypes.UpkeepResult{
			{Key: key1, PerformData: smallData, GasUsed: zeroBigInt, FastGasWei: zeroBigInt, LinkNative: zeroBigInt, CheckBlockNumber: 1239428374},
		}},
		{Name: "Large Perform Data", Data: []ktypes.UpkeepResult{
			{Key: key1, PerformData: largeData, GasUsed: zeroBigInt, FastGasWei: zeroBigInt, LinkNative: zeroBigInt, CheckBlockNumber: 1239428374},
		}},
		{Name: "Multiple Performs", Data: []ktypes.UpkeepResult{
			{Key: key1, PerformData: smallData, GasUsed: zeroBigInt, FastGasWei: zeroBigInt, LinkNative: zeroBigInt, CheckBlockNumber: 1239428374},
			{Key: key2, PerformData: largeData, GasUsed: zeroBigInt, FastGasWei: zeroBigInt, LinkNative: zeroBigInt, CheckBlockNumber: 1239428374},
			{Key: key3, PerformData: noData, GasUsed: zeroBigInt, FastGasWei: zeroBigInt, LinkNative: zeroBigInt, CheckBlockNumber: 1239428375},
		}},
	}

	for _, test := range tests {
		b.Run(test.Name, func(b *testing.B) {
			for n := 0; n < b.N; n++ {
				b.StartTimer()
				_, err := encoder.EncodeReport(test.Data)
				b.StopTimer()

				if err != nil {
					b.FailNow()
				}
			}
		})
	}
}
