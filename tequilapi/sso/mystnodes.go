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
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"sync"

	"github.com/mysteriumnetwork/node/config"
	"github.com/mysteriumnetwork/node/eventbus"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/requests"
	"github.com/mysteriumnetwork/node/tequilapi/contract"
	"github.com/mysteriumnetwork/node/tequilapi/pkce"
	"github.com/pkg/errors"
)

// ErrRedirectMissing host can't be blank
var ErrRedirectMissing = errors.New("host must not be empty")

// ErrNoUnlockedIdentity no unlocked identity
var ErrNoUnlockedIdentity = errors.New("lastUnlockedIdentity must not be empty")

// ErrCodeVerifierMissing code verifier is missing
var ErrCodeVerifierMissing = errors.New("no code verifier generated")

// ErrAuthorizationGrantTokenMissing blank authorization token
var ErrAuthorizationGrantTokenMissing = errors.New("token must be set")

// ErrMystnodesAuthorizationFail authorization failed against mystnodes
var ErrMystnodesAuthorizationFail = errors.New("mystnodes SSO grant authorization verification failed")

type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// Mystnodes SSO support
type Mystnodes struct {
	baseUrl                   string
	ssoPath                   string
	signer                    identity.SignerFactory
	lastUnlockedIdentity      identity.Identity
	client                    httpClient
	lastCodeVerifierBase64url string
	lock                      sync.Mutex
}

// NewMystnodes constructor
func NewMystnodes(signer identity.SignerFactory, client httpClient) *Mystnodes {
	return &Mystnodes{
		baseUrl: config.GetString(config.FlagMMNAddress),
		ssoPath: "/login-sso",
		signer:  signer,
		client:  client,
	}
}

// Subscribe unlocked identity is required in order to sign request
func (m *Mystnodes) Subscribe(eventBus eventbus.EventBus) error {
	if err := eventBus.SubscribeAsync(identity.AppTopicIdentityUnlock, m.onIdentityUnlocked); err != nil {
		return err
	}
	return nil
}

func (m *Mystnodes) onIdentityUnlocked(ev identity.AppEventIdentityUnlock) {
	m.lastUnlockedIdentity = ev.ID
}

func (m *Mystnodes) message(info pkce.Info, redirectURL string) MystnodesMessage {
	return MystnodesMessage{
		CodeChallenge: info.CodeChallenge,
		Identity:      m.lastUnlockedIdentity.Address,
		RedirectURL:   redirectURL,
	}
}

func (m *Mystnodes) sign(msg []byte) (identity.Signature, error) {
	return m.signer(m.lastUnlockedIdentity).Sign(msg)
}

// SSOLink build SSO link to begin authentication via mystnodes.com
func (m *Mystnodes) SSOLink(redirectURL *url.URL) (*url.URL, error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	if redirectURL == nil {
		return nil, ErrRedirectMissing
	}

	if len(m.lastUnlockedIdentity.Address) == 0 {
		return nil, ErrNoUnlockedIdentity
	}

	u, err := url.Parse(m.baseUrl)
	if err != nil {
		return &url.URL{}, err
	}

	u = u.JoinPath(m.ssoPath)

	info, err := pkce.New(128)
	if err != nil {
		return nil, err
	}

	m.lastCodeVerifierBase64url = info.Base64URLCodeVerifier()
	messageJson, err := m.message(info, fmt.Sprint(redirectURL)).JSON()
	if err != nil {
		return &url.URL{}, err
	}

	signature, err := m.sign(messageJson)
	if err != nil {
		return &url.URL{}, err
	}

	q := u.Query()
	q.Set("message", base64.RawURLEncoding.EncodeToString(messageJson))
	q.Set("signature", base64.RawURLEncoding.EncodeToString(signature.Bytes()))
	u.RawQuery = q.Encode()

	return u, nil
}

func (m *Mystnodes) consumeCodeVerifier() {
	m.lastCodeVerifierBase64url = ""
}

// VerifyAuthorizationGrant verifies authorization grant against mystnodes.com using PKCE workflow
func (m *Mystnodes) VerifyAuthorizationGrant(authorizationGrantToken string) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	defer m.consumeCodeVerifier()

	if len(m.lastCodeVerifierBase64url) == 0 {
		return ErrCodeVerifierMissing
	}

	if len(authorizationGrantToken) == 0 {
		return ErrAuthorizationGrantTokenMissing
	}

	req, err := requests.NewPostRequest(config.GetString(config.FlagMMNAPIAddress), "auth/sso-verify-grant", contract.MystnodesSSOGrantVerificationRequest{
		AuthorizationGrant:    authorizationGrantToken,
		CodeVerifierBase64url: m.lastCodeVerifierBase64url,
	})
	if err != nil {
		return err
	}

	res, err := m.client.Do(req)
	if err != nil {
		return err
	}

	if res.StatusCode < 200 || res.StatusCode > 299 {
		return errors.Wrap(ErrMystnodesAuthorizationFail, fmt.Sprintf("mystnodes SSO grant verification responded with %d", res.StatusCode))
	}

	return nil
}
