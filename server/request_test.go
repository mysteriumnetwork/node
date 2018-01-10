package server

import (
	"bytes"
	"github.com/mysterium/node/identity"
	"github.com/stretchr/testify/assert"
	"io"
	"net/url"
	"testing"
)

type testPayload struct {
	Value string `json:"value"`
}

type mockedSigner struct {
	signatureToReturn identity.Signature
}

func (signer *mockedSigner) Sign(message []byte) (identity.Signature, error) {
	return signer.signatureToReturn, nil
}

func TestSignatureIsInsertedForSignedPost(t *testing.T) {

	signer := mockedSigner{identity.SignatureHex("deadbeef")}

	req, err := newSignedPostRequest("/post-path", testPayload{"abc"}, &signer)

	assert.NoError(t, err)
	assert.NotEmpty(t, req.Header.Get("Authorization"))
}

func TestDoGetContactsPassedValuesForUrl(t *testing.T) {
	mysteriumApiUrl = "http://testUrl"

	params := url.Values{}
	params["param1"] = []string{"value1"}
	params["param2"] = []string{"value2"}

	req, err := newGetRequest("get-path", params)

	assert.NoError(t, err)
	assert.Equal(t, "http://testUrl/get-path?param1=value1&param2=value2", req.URL.String())

}

func TestPayloadIsSerializedSuccessfullyForPostMethod(t *testing.T) {

	req, err := newPostRequest("post-path", testPayload{"abc"})

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
