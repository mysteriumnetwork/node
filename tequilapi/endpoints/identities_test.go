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
			"identitiesListDto":[{
				"id": "0x000000000000000000000000000000000000bEEF"
			},
			{
				"id": "0x000000000000000000000000000000000000000F"
			}]
		}`,
		resp.Body.String())
}

func newManager() identity.IdentityManagerInterface {
	idmFake := identity.NewIdentityManagerFake()
	idmFake.CreateNewIdentity("0x000000000000000000000000000000000000bEEF")
	idmFake.CreateNewIdentity("0x000000000000000000000000000000000000000F")
	return idmFake
}
