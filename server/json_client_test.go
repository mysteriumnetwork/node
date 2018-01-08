package server

import (
	"bytes"
	"github.com/mysterium/node/identity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"io"
	"net/http"
	"net/url"
	"testing"
)

type testPayload struct {
	Value string `json:"value"`
}

type mockedHttpTransport struct {
	recordedRequest  *http.Request
	responseToReturn *http.Response
}

type mockedSigner struct {
	recordedMessage   []byte
	signatureToReturn identity.Signature
}

type signedJsonClientTestContext struct {
	suite.Suite
	mockedHttpTransport *mockedHttpTransport
	mockedSigner        *mockedSigner
	jsonHttpClient      *jsonHttpClient
}

func (ctx *signedJsonClientTestContext) SetupTest() {
	ctx.mockedHttpTransport = &mockedHttpTransport{
		nil,
		&http.Response{
			Status:     "OK",
			StatusCode: 200,
		},
	}
	ctx.mockedSigner = &mockedSigner{}
	ctx.jsonHttpClient = &jsonHttpClient{
		"http://testUrl",
		ctx.mockedHttpTransport,
	}
}

func (ctx *signedJsonClientTestContext) TestSignatureIsInsertedForSignedPost() {

	ctx.mockedSigner.signatureToReturn = identity.SignatureHex("deadbeef") //valid hex :)

	_, err := ctx.jsonHttpClient.DoSignedPost("/post-path", testPayload{"abc"}, ctx.mockedSigner)

	assert.NoError(ctx.T(), err)
	assert.NotEmpty(ctx.T(), ctx.mockedHttpTransport.recordedRequest.Header.Get("Authorization"))
}

func (ctx *signedJsonClientTestContext) TestDoGetContactsPassedValuesForUrl() {
	params := url.Values{}
	params["param1"] = []string{"value1"}
	params["param2"] = []string{"value2"}

	_, err := ctx.jsonHttpClient.DoGet("get-path", params)

	assert.NoError(ctx.T(), err)
	assert.Equal(ctx.T(), "http://testUrl/get-path?param1=value1&param2=value2", ctx.mockedHttpTransport.recordedRequest.URL.String())

}

func (ctx *signedJsonClientTestContext) TestPayloadIsSerializedSuccessfullyForPostMethod() {

	_, err := ctx.jsonHttpClient.DoPost("post-path", testPayload{"abc"})

	assert.NoError(ctx.T(), err)

	bodyBytes := bytes.NewBuffer(nil)
	_, err = io.Copy(bodyBytes, ctx.mockedHttpTransport.recordedRequest.Body)
	assert.NoError(ctx.T(), err)

	assert.JSONEq(
		ctx.T(),
		`{
			"value" : "abc"
		}`,
		bodyBytes.String(),
	)
}

func (ctx *signedJsonClientTestContext) TestHttpErrorIsReportedAsErrorReturnValue() {

	ctx.mockedHttpTransport.responseToReturn = &http.Response{
		StatusCode: 400,
	}
	_, err := ctx.jsonHttpClient.DoGet("some-path", nil)
	assert.Error(ctx.T(), err)
}

func (doer *mockedHttpTransport) Do(req *http.Request) (*http.Response, error) {
	doer.recordedRequest = req
	doer.responseToReturn.Request = req
	return doer.responseToReturn, nil
}

func (signer *mockedSigner) Sign(message []byte) (identity.Signature, error) {
	return signer.signatureToReturn, nil
}

func TestSignedHttpDoerSuite(t *testing.T) {
	suite.Run(t, new(signedJsonClientTestContext))
}
