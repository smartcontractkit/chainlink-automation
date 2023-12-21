package config_test

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"testing"

	"github.com/smartcontractkit/libocr/offchainreporting2plus/ocr3types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/curve25519"

	"github.com/smartcontractkit/chainlink-automation/tools/simulator/config"
	ocr2keepers "github.com/smartcontractkit/chainlink-common/pkg/types/automation"
)

func TestOffchainKeyring_OffchainSign(t *testing.T) {
	keyring, err := config.NewOffchainKeyring(rand.Reader, rand.Reader)

	require.NoError(t, err)

	signature, err := keyring.OffchainSign([]byte("message"))

	require.NoError(t, err)
	assert.Greater(t, len(signature), 0, "signature must have a length greater than zero")

	signature2, _ := keyring.OffchainSign([]byte("message"))

	assert.Equal(t, signature, signature2, "signing the same message twice should return equal results")
}

func TestOffchainKeyring_OffchainPublicKey(t *testing.T) {
	keyring, err := config.NewOffchainKeyring(rand.Reader, rand.Reader)

	require.NoError(t, err)

	pubkey := keyring.OffchainPublicKey()

	var compare [ed25519.PublicKeySize]byte

	assert.NotEqual(t, compare, pubkey, "public key should not be empty bytes")
}

func TestOffchainKeyring_ConfigEncryptionPublicKey(t *testing.T) {
	keyring, err := config.NewOffchainKeyring(rand.Reader, rand.Reader)

	require.NoError(t, err)

	pubkey := keyring.ConfigEncryptionPublicKey()

	var compare [curve25519.PointSize]byte

	assert.NotEqual(t, compare, pubkey, "public key should not be empty bytes")
}

func TestEvmKeyring(t *testing.T) {
	keyring, err := config.NewEVMKeyring(rand.Reader)
	keyring2, _ := config.NewEVMKeyring(rand.Reader)

	require.NoError(t, err)

	assert.NotEmpty(t, keyring.PublicKey(), "public key bytes should not be empty")
	assert.NotEmpty(t, keyring.PKString(), "public key string should not be empty")

	digest := sha256.Sum256([]byte("message"))
	round := uint64(10_000)
	report := ocr3types.ReportWithInfo[ocr2keepers.AutomationReportInfo]{
		Report: []byte("report"),
		Info:   ocr2keepers.AutomationReportInfo{},
	}

	signature, err := keyring.Sign(digest, round, report)

	require.NoError(t, err)
	require.NotEmpty(t, signature, "signature bytes should not be empty")

	verified := keyring2.Verify(keyring.PublicKey(), digest, round, report, signature)

	assert.True(t, verified, "keyring 2 must be able to verify signature of keyring 1")
}

func TestEvmKeyring_Encode(t *testing.T) {
	keyring, err := config.NewEVMKeyring(rand.Reader)
	keyring2, _ := config.NewEVMKeyring(rand.Reader)

	require.NoError(t, err)

	encoded, err := keyring.Marshal()

	require.NoError(t, err)
	require.NoError(t, keyring2.Unmarshal(encoded))

	assert.Equal(t, keyring.PKString(), keyring2.PKString())
}
