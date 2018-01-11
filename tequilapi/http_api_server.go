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

func NewServer(address string, port int, handler http.Handler) ApiServer {
	server := apiServer{
		make(chan error, 1),
		ApplyCors(handler),
		fmt.Sprintf("%s:%d", address, port),
		nil}
	return &server
}

func (server *apiServer) Stop() {
	server.listener.Close()
}

func (server *apiServer) Wait() error {
	return <-server.errorChannel
}

func (server *apiServer) Port() (int, error) {
	if server.listener == nil {
		return 0, errors.New("not bound")
	}
	return extractBoundPort(server.listener)
}

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
