package endpoints

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"bytes"
	"github.com/mysterium/node/identity"
	"github.com/stretchr/testify/assert"
)

var identityUrl = "/identities/123/registration"

func TestRegisterExistingIdentityRequest(t *testing.T) {
	req, err := http.NewRequest(
		http.MethodPut,
		identityUrl,
		bytes.NewBufferString(
			`{
				"registered": "false"
			}`))
	assert.Nil(t, err)
	resp := httptest.NewRecorder()

	mockIdm := identity.NewIdentityManagerFake()
	handlerFunc := NewIdentitiesEndpoint(mockIdm).Register
	handlerFunc(resp, req, nil)

}

func TestRegisterIdentitySuccess(t *testing.T) {
	req, err := http.NewRequest(
		http.MethodPut,
		identityUrl,
		bytes.NewBufferString(
			`{
				"registered": "true"
			}`))
	assert.Nil(t, err)
	resp := httptest.NewRecorder()

	mockIdm := identity.NewIdentityManagerFake()
	handlerFunc := NewIdentitiesEndpoint(mockIdm).Register
	handlerFunc(resp, req, nil)

}

func TestCreateNewIdentity(t *testing.T) {
	req, err := http.NewRequest(
		http.MethodPost,
		"/identities",
		bytes.NewBufferString(
			`{
				"password": "mypass"
			}`))
	assert.Nil(t, err)

	resp := httptest.NewRecorder()

	mockIdm := identity.NewIdentityManagerFake()
	handlerFunc := NewIdentitiesEndpoint(mockIdm).Create
	handlerFunc(resp, req, nil)

	assert.JSONEq(
		t,
		`
            {
                "id":"0x000000000000000000000000000000000000bEEF"
            }
        `,
		resp.Body.String(),
	)
}

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
