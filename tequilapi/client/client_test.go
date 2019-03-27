package client

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

const errorMessage = `
{
	"message" : "me haz faild"
}
`

func TestConnectionErrorIsReturnedByClientInsteadOfDoubleParsing(t *testing.T) {
	responseBody := &trackingCloser{
		Reader: strings.NewReader(errorMessage),
	}

	client := Client{
		http: &httpClient{
			http: onAnyRequestReturn(&http.Response{
				Status:     "Internal server error",
				StatusCode: 500,
				Body:       responseBody,
			}),
			baseURL:   "http://test-api-whatever",
			logPrefix: "test prefix ",
			ua:        "test-agent",
		},
	}

	_, err := client.Connect("consumer", "provider", "service", ConnectOptions{})
	assert.Error(t, err)
	assert.Equal(t, errors.New("server response invalid: Internal server error (http://test-api-whatever/connection). Possible error: me haz faild"), err)
	//when doing http request, response body should always be closed by client - otherwise persistent connections are leaking
	assert.True(t, responseBody.Closed)
}

type requestDoer func(req *http.Request) (*http.Response, error)

func (f requestDoer) Do(req *http.Request) (*http.Response, error) {
	return f(req)
}

func onAnyRequestReturn(response *http.Response) requestDoer {
	return func(req *http.Request) (*http.Response, error) {
		response.Request = req
		return response, nil
	}
}

type trackingCloser struct {
	io.Reader
	Closed bool
}

func (tc *trackingCloser) Close() error {
	tc.Closed = true
	return nil
}

var _ io.ReadCloser = (*trackingCloser)(nil)
