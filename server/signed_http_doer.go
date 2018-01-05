package server

import (
	"bytes"
	"github.com/mysterium/node/identity"
	"io"
	"net/http"
)

const (
	authenticationHeaderName = "Authorization"
	authenticationSchemaName = "Signature"
)

type signedHttpDoer struct {
	original HttpDoer
	signer   identity.Signer
}

func WithRequestSignatures(originalDoer HttpDoer, requestSigner identity.Signer) HttpDoer {
	return signedHttpDoer{
		originalDoer,
		requestSigner,
	}
}

func (shd signedHttpDoer) Do(req *http.Request) (*http.Response, error) {
	if req.Body == nil {
		//skip requests which has no body at the moment (GET , DELETE etc.)
		return shd.original.Do(req)
	}
	//unfortunatelly body signatures will work if and only if req contains byte.buffers as body (in case of other readers it will fail)
	//as we have no way to intercept reader, copy/calc hash and then pass reader unaffected to underlying transport
	//as a result - if body reader is consumed by us, original http request executor will have nothing to read and will send empty body
	bodyReader, err := req.GetBody()
	if err != nil {
		//with respect to original http client we also close requests original body
		req.Body.Close()
		return nil, err
	}
	defer bodyReader.Close()

	signature, err := signRequest(bodyReader, shd.signer)
	if err != nil {
		req.Body.Close()
		return nil, err
	}
	req.Header.Add(authenticationHeaderName, authenticationSchemaName+" "+signature.Base64())
	return shd.original.Do(req)
}

func signRequest(reader io.Reader, signer identity.Signer) (identity.Signature, error) {
	bodyBuffer := bytes.Buffer{}
	_, err := io.Copy(&bodyBuffer, reader)
	if err != nil {
		return identity.Signature{}, err
	}
	return signer.Sign(bodyBuffer.Bytes())
}
