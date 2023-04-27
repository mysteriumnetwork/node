package sso

import (
	"encoding/base64"
	"encoding/json"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/tequilapi/pkce"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"io"
	"net/http"
	"strings"
	"testing"
)

var signerFactory = func(id identity.Identity) identity.Signer {
	return &mockSignerFactory{}
}

type mockSignerFactory struct {
}

func (s *mockSignerFactory) Sign(message []byte) (identity.Signature, error) {
	return identity.SignatureBytes([]byte("pretty signature")), nil
}

func TestMystnodesSSOLink(t *testing.T) {
	// given
	sso := NewMystnodes(signerFactory, nil)
	sso.lastUnlockedIdentity = identity.Identity{Address: "0x1"}

	// expect
	r, err := sso.SSOLink("local:6969")
	assert.NoError(t, err)

	assert.Equal(t, r.Host, "")
	assert.NotEmpty(t, r.Query().Get("message"))

	decodedMessage, err := base64.RawURLEncoding.DecodeString(r.Query().Get("message"))
	assert.NoError(t, err)

	var mm MystnodesMessage
	err = json.Unmarshal(decodedMessage, &mm)
	assert.NoError(t, err)
	assert.Equal(t, sso.lastUnlockedIdentity.Address, mm.Identity)
	assert.Equal(t, "http://local:6969/#/auth-sso", mm.RedirectURL)
	assert.NotEmpty(t, mm.CodeChallenge)

	codeVerifierBytes, err := base64.RawURLEncoding.DecodeString(sso.lastCodeVerifierBase64url)
	assert.NoError(t, err)
	assert.True(t, pkce.ChallengeSHA256(string(codeVerifierBytes)) == mm.CodeChallenge)

	decodedSignatureBytes, err := base64.RawURLEncoding.DecodeString(r.Query().Get("signature"))
	assert.NoError(t, err)
	assert.Equal(t, "pretty signature", string(decodedSignatureBytes))
}

func TestMystnodesSSOLinkFail(t *testing.T) {
	// given
	sso := NewMystnodes(signerFactory, nil)

	// expect
	_, err := sso.SSOLink("")
	assert.ErrorIs(t, err, ErrHostMissing)

	_, err = sso.SSOLink("present")
	assert.ErrorIs(t, err, ErrNoUnlockedIdentity)
}

func TestMystnodesSSOGrantVerification(t *testing.T) {
	// when
	sso := NewMystnodes(signerFactory, newHttpClientMock(&http.Response{StatusCode: 200, Status: "200 OK", Body: &readCloser{Reader: strings.NewReader("{}")}}))
	sso.lastUnlockedIdentity = identity.Identity{Address: "0x1"}
	_, err := sso.SSOLink("not_important")
	assert.NoError(t, err)

	// then
	err = sso.VerifyAuthorizationGrant("auth_grant")
	assert.NoError(t, err)

	// when
	sso = NewMystnodes(signerFactory, newHttpClientMock(nil))
	sso.lastUnlockedIdentity = identity.Identity{Address: "0x1"}
	_, err = sso.SSOLink("not_important")
	assert.NoError(t, err)

	// then
	err = sso.VerifyAuthorizationGrant("auth_grant")
	assert.Error(t, err)
}

type httpClientMock struct {
	req *http.Request

	res *http.Response
}

func newHttpClientMock(res *http.Response) *httpClientMock {
	return &httpClientMock{
		res: res,
	}
}

func (m *httpClientMock) Do(req *http.Request) (*http.Response, error) {
	m.req = req
	if m.res == nil {
		return nil, errors.New("Oops")
	}
	return m.res, nil
}

func (m *httpClientMock) stub(res *http.Response) {
	m.res = res
}

type readCloser struct {
	io.Reader
	Closed bool
}

func (tc *readCloser) Close() error {
	tc.Closed = true
	return nil
}

var _ io.ReadCloser = (*readCloser)(nil)
