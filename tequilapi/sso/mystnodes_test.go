/*
 * Copyright (C) 2023 The "MysteriumNetwork/node" Authors.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */
package sso

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/tequilapi/pkce"

	"encoding/base64"
	"encoding/json"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
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
	redirect, err := url.Parse("http://local:6969/#/auth-sso")
	assert.NoError(t, err)

	r, err := sso.SSOLink(redirect)
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
	_, err := sso.SSOLink(nil)
	assert.ErrorIs(t, err, ErrRedirectMissing)

	_, err = sso.SSOLink(&url.URL{})
	assert.ErrorIs(t, err, ErrNoUnlockedIdentity)
}

func TestMystnodesSSOGrantVerification(t *testing.T) {
	// given
	redirect, err := url.Parse("http://not_important:6969/#/auth-sso")
	assert.NoError(t, err)

	// when
	sso := NewMystnodes(signerFactory, newHttpClientMock(&http.Response{StatusCode: 200, Status: "200 OK", Body: &readCloser{Reader: strings.NewReader(`{"walletAddress": "0x111", "apiKey": "xxx", "isEligibleForFreeRegistration": true}`)}}))
	sso.lastUnlockedIdentity = identity.Identity{Address: "0x1"}
	_, err = sso.SSOLink(redirect)
	assert.NoError(t, err)

	// then
	vi, err := sso.VerifyAuthorizationGrant("auth_grant", DefaultVerificationOptions)
	assert.NoError(t, err)
	assert.Equal(t, true, vi.IsEligibleForFreeRegistration)
	assert.Equal(t, "0x111", vi.WalletAddress)
	assert.Equal(t, "xxx", vi.APIkey)

	// when
	sso = NewMystnodes(signerFactory, newHttpClientMock(nil))
	sso.lastUnlockedIdentity = identity.Identity{Address: "0x1"}
	_, err = sso.SSOLink(redirect)
	assert.NoError(t, err)

	// then
	_, err = sso.VerifyAuthorizationGrant("auth_grant", DefaultVerificationOptions)
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
