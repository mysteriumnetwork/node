package endpoints

import (
	"net/http/httptest"
	"testing"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/julienschmidt/httprouter"
	"github.com/mysterium/node/identity"
	"github.com/stretchr/testify/assert"
)

func TestListIdentities(t *testing.T) {

	req := httptest.NewRequest("GET", "/irrelevant", nil)
	resp := httptest.NewRecorder()

	mockIdm := newManager("0x000000000000000000000000000000000000000A")
	handlerFunc := ListIdentitiesEndpointFactory(mockIdm).ListIdentities
	handlerFunc(resp, req, httprouter.Params{})

	assert.JSONEq(
		t,
		`{
            "identities":["0x000000000000000000000000000000000000000A"]
        }`,
		resp.Body.String())
}

func newManager(accountValue string) *identity.IdentityManager {
	return &identity.IdentityManager{
		KeystoreManager: &identity.KeyStoreFake{
			AccountsMock: []accounts.Account{
				identity.IdentityToAccount(accountValue),
			},
		},
	}
}
