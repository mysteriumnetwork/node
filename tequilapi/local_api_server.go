package tequilapi

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
)

type ApiServer interface {
	Port() int
	Wait()
	Stop()
}

type localApiServer struct {
	listener  net.Listener
	stopped   sync.WaitGroup
	bountPort int
}

func CreateNew(address string, port int) (ApiServer, error) {
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

	wg := sync.WaitGroup{}
	wg.Add(1)

	server := localApiServer{listener, wg, boundPort}
	go server.handleHttpRequests()
	return &server, nil
}

func (server *localApiServer) Stop() {
	server.listener.Close()
}

func (server *localApiServer) Wait() {
	server.stopped.Wait()
}

func (server *localApiServer) Port() int {
	return server.bountPort
}

func (server *localApiServer) handleHttpRequests() {
	http.Serve(server.listener, nil)
	server.stopped.Done()
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
