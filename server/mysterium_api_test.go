package server

import (
	"github.com/stretchr/testify/assert"
	"net"
	"net/http"
	"testing"
	"time"
)

func TestHttpTransportDoesntBlockForeverIfServerFailsToSendAnyResponse(t *testing.T) {

	address, err := createHTTPServer(func(writer http.ResponseWriter, request *http.Request) {
		select {} //infinite loop with no response to client
	})

	transport := newHTTPTransport(50 * time.Millisecond)
	req, err := http.NewRequest(http.MethodGet, "http://"+address+"/", nil)
	assert.NoError(t, err)

	completed := make(chan error)
	go func() {
		_, err := transport.Do(req)
		completed <- err
	}()

	select {
	case err := <-completed:
		netError, ok := err.(net.Error)
		assert.True(t, ok)
		assert.True(t, netError.Timeout())
	case <-time.After(1000 * time.Millisecond):
		assert.Fail(t, "failed request expected")

	}
}

func createHTTPServer(handlerFunc http.HandlerFunc) (address string, err error) {
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return
	}

	go http.Serve(listener, handlerFunc)
	return listener.Addr().String(), nil
}
