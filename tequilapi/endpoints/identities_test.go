package endpoints

import (
	"net/http/httptest"
	"testing"

	"github.com/mysterium/node/identity"
	"github.com/stretchr/testify/assert"
)

func TestListIdentities(t *testing.T) {

	req := httptest.NewRequest("GET", "/irrelevant", nil)
	resp := httptest.NewRecorder()

	mockIdm := identity.NewIdentityManagerFake()
	handlerFunc := NewIdentitiesEndpoint(mockIdm).List
	handlerFunc(resp, req, nil)

	assert.JSONEq(
		t,
		`[
            {
                "id": "0x000000000000000000000000000000000000bEEF"
            },
            {
                "id": "0x000000000000000000000000000000000000bEEF"
            }
        ]`,
		resp.Body.String(),
	)
}
