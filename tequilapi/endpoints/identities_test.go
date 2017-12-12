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

	mockIdm := newManager()
	handlerFunc := NewIdentitiesEndpoint(mockIdm).List
	handlerFunc(resp, req, httprouter.Params{})

	assert.JSONEq(
		t,
		`{
			"identities":[{
				"id": "0x000000000000000000000000000000000000bEEF"
			},
			{
				"id": "0x000000000000000000000000000000000000000F"
			}]
		}`,
		resp.Body.String())
}

func newManager() identity.IdentityManagerInterface {
	keystoreFake := &identity.KeyStoreFake{}
	keystoreFake.NewAccount("0x000000000000000000000000000000000000bEEF")
	keystoreFake.NewAccount("0x000000000000000000000000000000000000000F")
	return identity.NewIdentityManager(keystoreFake)
}
