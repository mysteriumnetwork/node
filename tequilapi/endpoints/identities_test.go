package endpoints

import (
	"net/http/httptest"
	"testing"

	"github.com/julienschmidt/httprouter"
	"github.com/mysterium/node/identity"
	"github.com/stretchr/testify/assert"
)

func TestListIdentities(t *testing.T) {

	req := httptest.NewRequest("GET", "/irrelevant", nil)
	resp := httptest.NewRecorder()

	mockIdm := newManager("0x000000000000000000000000000000000000bEEF")
	handlerFunc := IdentityHandlers(mockIdm).Get
	handlerFunc(resp, req, httprouter.Params{})

	assert.JSONEq(
		t,
		`{
			"identities":[{
				"id": "0x000000000000000000000000000000000000bEEF"
			}]
		}`,
		resp.Body.String())
}

func newManager(accountValue string) *identity.IdentityManager {
	keystoreFake := &identity.KeyStoreFake{}
	keystoreFake.NewAccount(accountValue)
	return identity.NewIdentityManager(keystoreFake)
}
