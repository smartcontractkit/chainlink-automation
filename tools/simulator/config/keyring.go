package config

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/ed25519"
	"encoding/binary"
	"io"

	"golang.org/x/crypto/curve25519"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/secp256k1"
	"github.com/smartcontractkit/libocr/offchainreporting2plus/ocr3types"
	"github.com/smartcontractkit/libocr/offchainreporting2plus/types"

	"github.com/smartcontractkit/chainlink-automation/pkg/v3/plugin"
)

var _ types.OffchainKeyring = &OffchainKeyring{}

// OffchainKeyring contains the secret keys needed for the OCR nodes to share secrets
// and perform aggregation.
//
// This is currently an ed25519 signing key and a separate encryption key.
//
// All its functions should be thread-safe.
type OffchainKeyring struct {
	signingKey    ed25519.PrivateKey
	encryptionKey [curve25519.ScalarSize]byte
}

func NewOffchainKeyring(encryptionMaterial, signingMaterial io.Reader) (*OffchainKeyring, error) {
	_, signingKey, err := ed25519.GenerateKey(signingMaterial)
	if err != nil {
		return nil, err
	}

	encryptionKey := [curve25519.ScalarSize]byte{}
	_, err = encryptionMaterial.Read(encryptionKey[:])
	if err != nil {
		return nil, err
	}

	keyring := &OffchainKeyring{
		signingKey:    signingKey,
		encryptionKey: encryptionKey,
	}
	_, err = keyring.configEncryptionPublicKey()
	if err != nil {
		return nil, err
	}
	return keyring, nil
}

// OffchainSign signs message using private key
func (k *OffchainKeyring) OffchainSign(msg []byte) (signature []byte, err error) {
	return ed25519.Sign(ed25519.PrivateKey(k.signingKey), msg), nil
}

// ConfigDiffieHellman returns the shared point obtained by multiplying someone's
// public key by a secret scalar ( in this case, the offchain key ring's encryption key.)
func (k *OffchainKeyring) ConfigDiffieHellman(point [curve25519.PointSize]byte) ([curve25519.PointSize]byte, error) {
	p, err := curve25519.X25519(k.encryptionKey[:], point[:])
	if err != nil {
		return [curve25519.PointSize]byte{}, err
	}
	sharedPoint := [ed25519.PublicKeySize]byte{}
	copy(sharedPoint[:], p)
	return sharedPoint, nil
}

// OffchainPublicKey returns the public component of this offchain keyring.
func (k *OffchainKeyring) OffchainPublicKey() types.OffchainPublicKey {
	var offchainPubKey [ed25519.PublicKeySize]byte
	copy(offchainPubKey[:], k.signingKey.Public().(ed25519.PublicKey)[:])
	return offchainPubKey
}

// ConfigEncryptionPublicKey returns config public key
func (k *OffchainKeyring) ConfigEncryptionPublicKey() types.ConfigEncryptionPublicKey {
	cpk, _ := k.configEncryptionPublicKey()
	return cpk
}

func (k *OffchainKeyring) configEncryptionPublicKey() (types.ConfigEncryptionPublicKey, error) {
	rv, err := curve25519.X25519(k.encryptionKey[:], curve25519.Basepoint)
	if err != nil {
		return [curve25519.PointSize]byte{}, err
	}
	var rvFixed [curve25519.PointSize]byte
	copy(rvFixed[:], rv)
	return rvFixed, nil
}

var curve = secp256k1.S256()

var _ ocr3types.OnchainKeyring[plugin.AutomationReportInfo] = &EvmKeyring{}

type EvmKeyring struct {
	privateKey ecdsa.PrivateKey
}

func NewEVMKeyring(material io.Reader) (*EvmKeyring, error) {
	ecdsaKey, err := ecdsa.GenerateKey(curve, material)
	if err != nil {
		return nil, err
	}
	return &EvmKeyring{privateKey: *ecdsaKey}, nil
}

// PublicKey returns the address of the public key not the public key itself
func (k *EvmKeyring) PublicKey() types.OnchainPublicKey {
	return k.signingAddress().Bytes()
}

// XXX: PublicKey returns the address of the public key not the public key itself
func (k *EvmKeyring) PKString() string {
	return k.signingAddress().String()
}

func (k *EvmKeyring) reportToSigData(digest types.ConfigDigest, v uint64, r ocr3types.ReportWithInfo[plugin.AutomationReportInfo]) []byte {
	rawRepctx := [3][32]byte{}

	// first is the digest
	copy(rawRepctx[0][:], digest[:])

	// then the round index
	binary.BigEndian.PutUint64(rawRepctx[1][32-8:], v)

	sigData := crypto.Keccak256(r.Report)
	sigData = append(sigData, rawRepctx[0][:]...)
	sigData = append(sigData, rawRepctx[1][:]...)
	sigData = append(sigData, rawRepctx[2][:]...)

	return crypto.Keccak256(sigData)
}

func (k *EvmKeyring) Sign(digest types.ConfigDigest, v uint64, r ocr3types.ReportWithInfo[plugin.AutomationReportInfo]) ([]byte, error) {
	return crypto.Sign(k.reportToSigData(digest, v, r), &k.privateKey)
}

func (k *EvmKeyring) Verify(publicKey types.OnchainPublicKey, digest types.ConfigDigest, v uint64, r ocr3types.ReportWithInfo[plugin.AutomationReportInfo], signature []byte) bool {
	hash := k.reportToSigData(digest, v, r)
	authorPubkey, err := crypto.SigToPub(hash, signature)
	if err != nil {
		return false
	}
	authorAddress := crypto.PubkeyToAddress(*authorPubkey)
	return bytes.Equal(publicKey[:], authorAddress[:])
}

func (k *EvmKeyring) MaxSignatureLength() int {
	return 65
}

func (k *EvmKeyring) signingAddress() common.Address {
	return crypto.PubkeyToAddress(*(&k.privateKey).Public().(*ecdsa.PublicKey))
}

func (k *EvmKeyring) Marshal() ([]byte, error) {
	return crypto.FromECDSA(&k.privateKey), nil
}

func (k *EvmKeyring) Unmarshal(in []byte) error {
	privateKey, err := crypto.ToECDSA(in)
	if err != nil {
		return err
	}

	k.privateKey = *privateKey

	return nil
}
