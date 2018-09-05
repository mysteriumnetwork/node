/*
 * Copyright (C) 2017 The "MysteriumNetwork/node" Authors.
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

package requests

import (
	"bytes"
	"io"
	"net/url"
	"testing"

	"github.com/mysteriumnetwork/node/identity"
	"github.com/stretchr/testify/assert"
)

type testPayload struct {
	Value string `json:"value"`
}

type mockedSigner struct {
	signatureToReturn identity.Signature
}

var testRequestApiUrl = "http://testUrl"

func (signer *mockedSigner) Sign(message []byte) (identity.Signature, error) {
	return signer.signatureToReturn, nil
}

func TestSignatureIsInsertedForSignedPost(t *testing.T) {

	signer := mockedSigner{identity.SignatureBase64("deadbeef")}

	req, err := NewSignedPostRequest(testRequestApiUrl, "/post-path", testPayload{"abc"}, &signer)
	assert.NoError(t, err)
	assert.Equal(t, req.Header.Get("Authorization"), "Signature deadbeef")
}

func TestDoGetContactsPassedValuesForUrl(t *testing.T) {

	params := url.Values{}
	params["param1"] = []string{"value1"}
	params["param2"] = []string{"value2"}

	req, err := NewGetRequest(testRequestApiUrl, "get-path", params)

	assert.NoError(t, err)
	assert.Equal(t, "http://testUrl/get-path?param1=value1&param2=value2", req.URL.String())

}

func TestPayloadIsSerializedSuccessfullyForPostMethod(t *testing.T) {

	req, err := NewPostRequest(testRequestApiUrl, "post-path", testPayload{"abc"})

	assert.NoError(t, err)

	bodyBytes := bytes.NewBuffer(nil)
	_, err = io.Copy(bodyBytes, req.Body)
	assert.NoError(t, err)

	assert.JSONEq(
		t,
		`{
			"value" : "abc"
		}`,
		bodyBytes.String(),
	)
}
