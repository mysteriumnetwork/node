package tequilapi

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
)

type ApiServer interface {
	Port() int
	Wait() error
	StartServing(handler http.Handler)
	Stop()
}

type apiServer struct {
	listener     net.Listener
	errorChannel chan error
	boundPort    int
}

func NewServer(address string, port int) (ApiServer, error) {
	var err error
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", address, port))
	if err != nil {
		return nil, err
	}

	boundPort, err := extractBoundPort(listener)
	if err != nil {
		listener.Close()
		return nil, err
	}

	server := apiServer{listener, make(chan error, 1), boundPort}
	return &server, nil
}

func (server *apiServer) Stop() {
	server.listener.Close()
}

func (server *apiServer) Wait() error {
	return <-server.errorChannel
}

func (server *apiServer) Port() int {
	return server.boundPort
}

func (server *apiServer) StartServing(handler http.Handler) {
	go server.serve(handler)
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
