package mocks

import (
	"github.com/smartcontractkit/libocr/offchainreporting2/types"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/curve25519"
)

type MockOffchainKeyring struct {
	mock.Mock
}

// OffchainSign returns an EdDSA-Ed25519 signature on msg produced using the
// standard library's ed25519.Sign function.
func (_m *MockOffchainKeyring) OffchainSign(msg []byte) (signature []byte, err error) {
	ret := _m.Mock.Called(msg)

	var r0 []byte
	if rf, ok := ret.Get(0).(func() []byte); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]byte)
		}
	}

	return r0, ret.Error(1)
}

// ConfigDiffieHellman multiplies point with the secret key (i.e. scalar)
// that ConfigEncryptionPublicKey corresponds to.
func (_m *MockOffchainKeyring) ConfigDiffieHellman(point [curve25519.PointSize]byte) (sharedPoint [curve25519.PointSize]byte, err error) {
	ret := _m.Mock.Called(point)

	var r0 [curve25519.PointSize]byte
	if rf, ok := ret.Get(0).(func() [curve25519.PointSize]byte); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([curve25519.PointSize]byte)
		}
	}

	return r0, ret.Error(1)
}

// OffchainPublicKey returns the public component of the keypair used in SignOffchain.
func (_m *MockOffchainKeyring) OffchainPublicKey() types.OffchainPublicKey {
	ret := _m.Mock.Called()

	var r0 types.OffchainPublicKey
	if rf, ok := ret.Get(0).(func() types.OffchainPublicKey); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(types.OffchainPublicKey)
		}
	}

	return r0
}

// ConfigEncryptionPublicKey returns the public component of the keypair used in ConfigDiffieHellman.
func (_m *MockOffchainKeyring) ConfigEncryptionPublicKey() types.ConfigEncryptionPublicKey {
	ret := _m.Mock.Called()

	var r0 types.ConfigEncryptionPublicKey
	if rf, ok := ret.Get(0).(func() types.ConfigEncryptionPublicKey); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(types.ConfigEncryptionPublicKey)
		}
	}

	return r0
}

type MockOnchainKeyring struct {
	mock.Mock
}

// PublicKey returns the public key of the keypair used by Sign.
func (_m *MockOnchainKeyring) PublicKey() types.OnchainPublicKey {
	ret := _m.Mock.Called()

	var r0 types.OnchainPublicKey
	if rf, ok := ret.Get(0).(func() types.OnchainPublicKey); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(types.OnchainPublicKey)
		}
	}

	return r0
}

// Sign returns a signature over ReportContext and Report.
func (_m *MockOnchainKeyring) Sign(rc types.ReportContext, r types.Report) (signature []byte, err error) {
	ret := _m.Mock.Called(rc, r)

	var r0 []byte
	if rf, ok := ret.Get(0).(func() []byte); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]byte)
		}
	}

	return r0, ret.Error(1)
}

// Verify verifies a signature over ReportContext and Report allegedly
// created from OnchainPublicKey.
//
// Implementations of this function must gracefully handle malformed or
// adversarially crafted inputs.
func (_m *MockOnchainKeyring) Verify(pk types.OnchainPublicKey, rc types.ReportContext, r types.Report, signature []byte) bool {
	ret := _m.Mock.Called(pk, rc, r, signature)

	var r0 bool
	if rf, ok := ret.Get(0).(func() bool); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(bool)
		}
	}

	return r0
}

// Maximum length of a signature
func (_m *MockOnchainKeyring) MaxSignatureLength() int {
	ret := _m.Mock.Called()

	var r0 int
	if rf, ok := ret.Get(0).(func() int); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(int)
		}
	}

	return r0
}
