/*
 * Copyright (C) 2017 The "MysteriumNetwork/node" Authors.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package management

import (
	"bufio"
	"net"
	"net/textproto"
	"sync"
	"time"

	"fmt"
	log "github.com/cihub/seelog"
	"io"
	"strings"
)

// Addr struct represents local address on which listener waits for incoming management connections
type Addr struct {
	IP   string
	Port int
}

// LocalhostOnRandomPort defines localhost address with randomly bound port
var LocalhostOnRandomPort = Addr{
	IP:   "127.0.0.1",
	Port: 0,
}

// String returns address string representation
func (addr *Addr) String() string {
	return fmt.Sprintf("%s:%d", addr.IP, addr.Port)
}

// Management structure represents connection and interface to openvpn management
type Management struct {
	BoundAddress Addr
	Connected    chan bool
	logPrefix    string

	middlewares []Middleware

	shutdownStarted chan bool
	shutdownWaiter  sync.WaitGroup
	closesOnce      sync.Once
}

// NewManagement creates new manager for given sock address, uses given log prefix for logging and takes a list of middlewares
func NewManagement(socketAddress Addr, logPrefix string, middlewares ...Middleware) *Management {
	return &Management{
		BoundAddress: socketAddress,
		Connected:    make(chan bool, 1),
		logPrefix:    logPrefix,

		middlewares: middlewares,

		shutdownStarted: make(chan bool),
		shutdownWaiter:  sync.WaitGroup{},
	}
}

// WaitForConnection method starts listener on bind address and returns "real" bound address (with port not zero) and
// channel which receives true when connection is accepted or false overwise (i.e. listener stop requested). It returns non nil
// error on any error condition
func (management *Management) WaitForConnection() error {
	log.Info(management.logPrefix, "Binding to socket: ", management.BoundAddress.String())

	listener, err := net.Listen("tcp", management.BoundAddress.String())
	if err != nil {
		log.Error(management.logPrefix, err)
		return err
	}

	netAddress := listener.Addr().(*net.TCPAddr)
	management.BoundAddress = Addr{
		netAddress.IP.String(),
		netAddress.Port,
	}

	log.Info(
		management.logPrefix,
		"Waiting for incoming connection on: ",
		management.BoundAddress.String(),
	)

	management.shutdownWaiter.Add(1)
	go management.listenForConnection(listener)

	return nil
}

// Stop initiates managemnt shutdown
func (management *Management) Stop() {
	log.Info(management.logPrefix, "Shutdown")
	management.closesOnce.Do(func() {
		close(management.shutdownStarted)
	})

	management.shutdownWaiter.Wait()

	log.Info(management.logPrefix, "Shutdown finished")
}

func (management *Management) listenForConnection(listener net.Listener) {
	defer management.shutdownWaiter.Done()
	defer listener.Close()

	connChannel := make(chan net.Conn, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			log.Critical(management.logPrefix, "Connection accept error: ", err)
			close(connChannel)
			return
		}
		connChannel <- conn
	}()

	select {
	case conn := <-connChannel:
		if conn != nil {
			management.Connected <- true
			go management.serveNewConnection(conn)
		}
	case <-management.shutdownStarted:
		management.Connected <- false
	}
}

func (management *Management) serveNewConnection(netConn net.Conn) {
	log.Info(management.logPrefix, "New connection started")
	defer netConn.Close()

	cmdOutputChannel := make(chan string)
	//make event channel buffered, so we can assure all middlewares are started before first event is delivered to middleware
	eventChannel := make(chan string, 100)
	connection := newChannelConnection(netConn, cmdOutputChannel)

	outputConsuming := sync.WaitGroup{}
	outputConsuming.Add(1)
	go func() {
		defer outputConsuming.Done()
		management.consumeOpenvpnConnectionOutput(netConn, cmdOutputChannel, eventChannel)
	}()

	management.startMiddlewares(connection)
	defer management.stopMiddlewares(connection)

	//start delivering events to middlewares
	go management.deliverOpenvpnManagementEvents(eventChannel)
	//block until output consumption is done - usually when connection is closed by openvpn process
	outputConsuming.Wait()
}

func (management *Management) startMiddlewares(connection CommandWriter) {
	for _, middleware := range management.middlewares {
		err := middleware.Start(connection)
		if err != nil {
			//TODO what we should do with errors on middleware start? Stop already running, close cmdWriter, bailout?
			//at least log errors for now
			log.Error(management.logPrefix, "Middleware startup error: ", err)
		}
	}
}

func (management *Management) stopMiddlewares(connection CommandWriter) {
	for _, middleware := range management.middlewares {
		err := middleware.Stop(connection)
		if err != nil {
			//log error but do not stop cleaning process
			log.Warn(management.logPrefix, "Middleware stop error. ", err)
		}
	}
}

func (management *Management) consumeOpenvpnConnectionOutput(input io.Reader, outputChannel, eventChannel chan string) {
	reader := textproto.NewReader(bufio.NewReader(input))
	for {
		line, err := reader.ReadLine()
		if err != nil {
			log.Warn(management.logPrefix, "Connection failed to read. ", err)
			return
		}
		log.Debug(management.logPrefix, "Line received: ", line)

		output := outputChannel
		if strings.HasPrefix(line, ">") {
			output = eventChannel
		}

		// Try to deliver the message
		select {
		case output <- line:
		case <-time.After(time.Second):
			log.Error(management.logPrefix, "Failed to transport line: ", line)
		}
	}
}

func (management *Management) deliverOpenvpnManagementEvents(eventChannel chan string) {
	for {
		event := <-eventChannel
		log.Trace(management.logPrefix, "Line delivering: ", event)

		lineConsumed := false
		for _, middleware := range management.middlewares {
			consumed, err := middleware.ConsumeLine(event)
			if err != nil {
				log.Error(management.logPrefix, "Failed to deliver event: ", event, ". ", err)
			}

			lineConsumed = lineConsumed || consumed
		}
		if !lineConsumed {
			log.Trace(management.logPrefix, "Line not delivered: ", event)
		}
	}
}
