package config

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/ed25519"
	"io"

	"golang.org/x/crypto/curve25519"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/secp256k1"
	"github.com/smartcontractkit/libocr/offchainreporting2plus/chains/evmutil"
	"github.com/smartcontractkit/libocr/offchainreporting2plus/types"
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

	ok := &OffchainKeyring{
		signingKey:    signingKey,
		encryptionKey: encryptionKey,
	}
	_, err = ok.configEncryptionPublicKey()
	if err != nil {
		return nil, err
	}
	return ok, nil
}

// OffchainSign signs message using private key
func (ok *OffchainKeyring) OffchainSign(msg []byte) (signature []byte, err error) {
	return ed25519.Sign(ed25519.PrivateKey(ok.signingKey), msg), nil
}

// ConfigDiffieHellman returns the shared point obtained by multiplying someone's
// public key by a secret scalar ( in this case, the offchain key ring's encryption key.)
func (ok *OffchainKeyring) ConfigDiffieHellman(point [curve25519.PointSize]byte) ([curve25519.PointSize]byte, error) {
	p, err := curve25519.X25519(ok.encryptionKey[:], point[:])
	if err != nil {
		return [curve25519.PointSize]byte{}, err
	}
	sharedPoint := [ed25519.PublicKeySize]byte{}
	copy(sharedPoint[:], p)
	return sharedPoint, nil
}

// OffchainPublicKey returns the public component of this offchain keyring.
func (ok *OffchainKeyring) OffchainPublicKey() types.OffchainPublicKey {
	var offchainPubKey [ed25519.PublicKeySize]byte
	copy(offchainPubKey[:], ok.signingKey.Public().(ed25519.PublicKey)[:])
	return offchainPubKey
}

// ConfigEncryptionPublicKey returns config public key
func (ok *OffchainKeyring) ConfigEncryptionPublicKey() types.ConfigEncryptionPublicKey {
	cpk, _ := ok.configEncryptionPublicKey()
	return cpk
}

func (ok *OffchainKeyring) configEncryptionPublicKey() (types.ConfigEncryptionPublicKey, error) {
	rv, err := curve25519.X25519(ok.encryptionKey[:], curve25519.Basepoint)
	if err != nil {
		return [curve25519.PointSize]byte{}, err
	}
	var rvFixed [curve25519.PointSize]byte
	copy(rvFixed[:], rv)
	return rvFixed, nil
}

var curve = secp256k1.S256()

var _ types.OnchainKeyring = &evmKeyring{}

type evmKeyring struct {
	privateKey ecdsa.PrivateKey
}

func NewEVMKeyring(material io.Reader) (*evmKeyring, error) {
	ecdsaKey, err := ecdsa.GenerateKey(curve, material)
	if err != nil {
		return nil, err
	}
	return &evmKeyring{privateKey: *ecdsaKey}, nil
}

// XXX: PublicKey returns the address of the public key not the public key itself
func (ok *evmKeyring) PublicKey() types.OnchainPublicKey {
	address := ok.signingAddress()
	return address[:]
}

// XXX: PublicKey returns the address of the public key not the public key itself
func (ok *evmKeyring) PKString() string {
	return ok.signingAddress().String()
}

func (ok *evmKeyring) reportToSigData(reportCtx types.ReportContext, report types.Report) []byte {
	rawReportContext := evmutil.RawReportContext(reportCtx)
	sigData := crypto.Keccak256(report)
	sigData = append(sigData, rawReportContext[0][:]...)
	sigData = append(sigData, rawReportContext[1][:]...)
	sigData = append(sigData, rawReportContext[2][:]...)
	return crypto.Keccak256(sigData)
}

func (ok *evmKeyring) Sign(reportCtx types.ReportContext, report types.Report) ([]byte, error) {
	return crypto.Sign(ok.reportToSigData(reportCtx, report), &ok.privateKey)

}

func (ok *evmKeyring) Verify(publicKey types.OnchainPublicKey, reportCtx types.ReportContext, report types.Report, signature []byte) bool {
	hash := ok.reportToSigData(reportCtx, report)
	authorPubkey, err := crypto.SigToPub(hash, signature)
	if err != nil {
		return false
	}
	authorAddress := crypto.PubkeyToAddress(*authorPubkey)
	return bytes.Equal(publicKey[:], authorAddress[:])
}

func (ok *evmKeyring) MaxSignatureLength() int {
	return 65
}

func (ok *evmKeyring) signingAddress() common.Address {
	return crypto.PubkeyToAddress(*(&ok.privateKey).Public().(*ecdsa.PublicKey))
}

func (ok *evmKeyring) Marshal() ([]byte, error) {
	return crypto.FromECDSA(&ok.privateKey), nil
}

func (ok *evmKeyring) Unmarshal(in []byte) error {
	privateKey, err := crypto.ToECDSA(in)
	if err != nil {
		return err
	}
	ok.privateKey = *privateKey
	return nil
}
