package server

import (
	"bytes"
	"github.com/mysterium/node/identity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"net/http"
	"testing"
)

type mockedHttpDoer struct {
	recordedRequest *http.Request
}

type mockedSigner struct {
	recordedMessage   []byte
	signatureToReturn identity.Signature
}

type signedHttpDoerTestContext struct {
	suite.Suite
	mockedOriginalDoer *mockedHttpDoer
	mockedSigner       *mockedSigner
	signedHttpDoer     *signedHttpDoer
}

func (ctx *signedHttpDoerTestContext) SetupTest() {
	ctx.mockedOriginalDoer = &mockedHttpDoer{}
	ctx.mockedSigner = &mockedSigner{}
	ctx.signedHttpDoer = &signedHttpDoer{
		ctx.mockedOriginalDoer,
		ctx.mockedSigner,
	}
}

func (ctx *signedHttpDoerTestContext) TestSignatureIsInsertedForRequestWithBody() {
	bodyAsString := `
		{
			"field" : "value"
		}
	`
	req, err := http.NewRequest(http.MethodPost, "/path", bytes.NewBufferString(bodyAsString))
	assert.NoError(ctx.T(), err)

	ctx.mockedSigner.signatureToReturn = identity.SignatureBytes([]byte{'a', 'b', 'c'})

	_, err = ctx.signedHttpDoer.Do(req)

	assert.NoError(ctx.T(), err)
	assert.Equal(ctx.T(), "Signature YWJj", ctx.mockedOriginalDoer.recordedRequest.Header.Get("Authorization"))
}

func (ctx *signedHttpDoerTestContext) TestSignatureIsNotPresentForRequestWithoutBody() {

	req, err := http.NewRequest(http.MethodGet, "/path", nil)
	assert.NoError(ctx.T(), err)

	_, err = ctx.signedHttpDoer.Do(req)

	assert.NoError(ctx.T(), err)
	assert.Empty(ctx.T(), ctx.mockedOriginalDoer.recordedRequest.Header.Get("Authorization"))
}

func (doer *mockedHttpDoer) Do(req *http.Request) (*http.Response, error) {
	doer.recordedRequest = req
	return nil, nil
}

func (signer *mockedSigner) Sign(message []byte) (identity.Signature, error) {
	return signer.signatureToReturn, nil
}

func TestSignedHttpDoerSuite(t *testing.T) {
	suite.Run(t, new(signedHttpDoerTestContext))
}
