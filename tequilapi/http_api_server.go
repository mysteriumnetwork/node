package tequilapi

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
)

// APIServer interface represents control methods for underlying http api server
type APIServer interface {
	Port() (int, error)
	Wait() error
	StartServing() error
	Stop()
}

type apiServer struct {
	errorChannel  chan error
	handler       http.Handler
	listenAddress string
	listener      net.Listener
}

// NewServer creates http api server for given addres port and http handler
func NewServer(address string, port int, handler http.Handler) APIServer {
	server := apiServer{
		make(chan error, 1),
		DisableCaching(ApplyCors(handler)),
		fmt.Sprintf("%s:%d", address, port),
		nil}
	return &server
}

// Stop method stops underlying http server
func (server *apiServer) Stop() {
	if server.listener == nil {
		return
	}
	server.listener.Close()
}

// Wait method waits for http server to finish handling requets (i.e. when Stop() was called)
func (server *apiServer) Wait() error {
	return <-server.errorChannel
}

// Port method returns bind port for given http server (useful when random port is used)
func (server *apiServer) Port() (int, error) {
	if server.listener == nil {
		return 0, errors.New("not bound")
	}
	return extractBoundPort(server.listener)
}

// StartServing starts http request serving
func (server *apiServer) StartServing() error {
	var err error
	server.listener, err = net.Listen("tcp", server.listenAddress)
	if err != nil {
		return err
	}
	go server.serve(server.handler)
	return nil
}

func (server *apiServer) serve(handler http.Handler) {
	server.errorChannel <- http.Serve(server.listener, handler)
}

func extractBoundPort(listener net.Listener) (int, error) {
	addr := listener.Addr()
	//it is possible that address could be x.x.x.x:y (IPv4) or [x:x:..:x]:y (IPv6) format
	//split by : and take the last one
	parts := strings.Split(addr.String(), ":")
	if len(parts) < 2 {
		return 0, errors.New("Unable to locate port of the following address: " + addr.String())
	}
	portAsString := parts[len(parts)-1]
	port, err := strconv.Atoi(portAsString)
	return port, err
}
