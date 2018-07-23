package registry

import (
	"github.com/MysteriumNetwork/payments/identity"
	"github.com/MysteriumNetwork/payments/registry"
	"github.com/ethereum/go-ethereum/common"
	"github.com/julienschmidt/httprouter"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

const (
	testPublicKeyPart1 = "0xFA001122334455667788990011223344556677889900112233445566778899AF"
	testPublicKeyPart2 = "0xDE001122334455667788990011223344556677889900112233445566778899AD"
)

func TestRegistrationEndpointReturnsRegistrationData(t *testing.T) {

	mockedDataProvider := &mockRegistrationDataProvider{}
	mockedDataProvider.RegistrationData = &registry.RegistrationData{
		PublicKey: registry.PublicKeyParts{
			Part1: common.FromHex(testPublicKeyPart1),
			Part2: common.FromHex(testPublicKeyPart2),
		},
		Signature: &identity.DecomposedSignature{
			R: [32]byte{1},
			S: [32]byte{2},
			V: 27,
		},
	}

	mockedStatusProvider := &mockRegistrationStatus{
		Registered: false,
	}

	endpoint := newRegistrationEndpoint(mockedDataProvider, mockedStatusProvider)

	req, err := http.NewRequest(
		http.MethodGet,
		"/notimportant",
		nil,
	)
	assert.NoError(t, err)

	resp := httptest.NewRecorder()

	endpoint.RegistrationData(
		resp,
		req,
		httprouter.Params{
			httprouter.Param{
				Key:   "id",
				Value: "0x1231323131",
			},
		},
	)

	assert.Equal(t, common.HexToAddress("0x1231323131"), mockedDataProvider.RecordedIdentity)

	assert.JSONEq(
		t,
		`{
			"Registered" : false,
            "PublicKey": {
				"Part1" : "0xfa001122334455667788990011223344556677889900112233445566778899af",
				"Part2" : "0xde001122334455667788990011223344556677889900112233445566778899ad"
			},
			"Signature": {
				"R": "0x0100000000000000000000000000000000000000000000000000000000000000",
				"S": "0x0200000000000000000000000000000000000000000000000000000000000000",
				"V": 27
			}
        }`,
		resp.Body.String(),
	)

}

func TestRegistrationEndpointReturnsOnlyRegistrationStatusForRegisteredIdentity(t *testing.T) {
	mockedRegistrationStatus := &mockRegistrationStatus{
		Registered: true,
	}

	mockedRegistrationDataProvider := &mockRegistrationDataProvider{}

	endpoint := newRegistrationEndpoint(mockedRegistrationDataProvider, mockedRegistrationStatus)

	req, err := http.NewRequest(
		http.MethodGet,
		"/notimportant",
		nil,
	)
	assert.NoError(t, err)

	resp := httptest.NewRecorder()

	endpoint.RegistrationData(
		resp,
		req,
		httprouter.Params{
			httprouter.Param{
				Key:   "id",
				Value: "0x1231323131",
			},
		},
	)

	assert.JSONEq(
		t,
		`{
			"Registered" : true
        }`,
		resp.Body.String(),
	)

}

type mockRegistrationStatus struct {
	Registered bool
}

func (m *mockRegistrationStatus) IsRegistered(identity common.Address) (bool, error) {
	return m.Registered, nil
}

type mockRegistrationDataProvider struct {
	RegistrationData *registry.RegistrationData
	RecordedIdentity common.Address
}

func (m *mockRegistrationDataProvider) ProvideRegistrationData(identity common.Address) (*registry.RegistrationData, error) {
	m.RecordedIdentity = identity
	return m.RegistrationData, nil
}
