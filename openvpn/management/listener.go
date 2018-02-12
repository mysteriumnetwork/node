package management

import (
	"bufio"
	"net"
	"net/textproto"
	"sync"
	"time"

	log "github.com/cihub/seelog"
	"io"
	"strings"
)

// Management structure represents connection and interface to openvpn management
type Management struct {
	socketAddress string
	logPrefix     string

	middlewares []Middleware

	listenerShutdownStarted chan bool
	listenerShutdownWaiter  sync.WaitGroup
	closesOnce              sync.Once
}

// NewManagement creates new manager for given sock address, uses given log prefix for logging and takes a list of middlewares
func NewManagement(socketAddress, logPrefix string, middlewares ...Middleware) *Management {
	return &Management{
		socketAddress: socketAddress,
		logPrefix:     logPrefix,

		middlewares: middlewares,

		listenerShutdownStarted: make(chan bool),
		listenerShutdownWaiter:  sync.WaitGroup{},
	}
}

func (management *Management) Start() error {
	log.Info(management.logPrefix, "Connecting to socket:", management.socketAddress)

	listener, err := net.Listen("unix", management.socketAddress)
	if err != nil {
		log.Error(management.logPrefix, err)
		return err
	}

	go management.waitForShutdown(listener)
	go management.waitForConnections(listener)

	return nil
}

func (management *Management) Stop() {
	log.Info(management.logPrefix, "Shutdown")
	management.closesOnce.Do(func() {
		close(management.listenerShutdownStarted)
	})

	management.listenerShutdownWaiter.Wait()
	log.Info(management.logPrefix, "Shutdown finished")
}

// SocketAddress returns management socket address
func (management *Management) SocketAddress() string {
	return management.socketAddress
}

func (management *Management) waitForShutdown(listener net.Listener) {
	<-management.listenerShutdownStarted
	listener.Close()
}

func (management *Management) waitForConnections(listener net.Listener) {
	management.listenerShutdownWaiter.Add(1)
	defer management.listenerShutdownWaiter.Done()

	for {
		connection, err := listener.Accept()
		if err != nil {
			select {
			case <-management.listenerShutdownStarted:
				log.Info(management.logPrefix, "Connection closed")
			default:
				log.Critical(management.logPrefix, "Connection accept error: ", err)
			}
			return
		}

		go management.serveNewConnection(connection)
	}
}

func (management *Management) serveNewConnection(netConn net.Conn) {
	log.Info(management.logPrefix, "New connection started")
	defer netConn.Close()

	cmdOutputChannel := make(chan string)
	eventChannel := make(chan string)
	connection := newSocketConnection(netConn, cmdOutputChannel)
	go management.deliverOpenvpnManagementEvents(eventChannel)

	outputConsuming := sync.WaitGroup{}
	outputConsuming.Add(1)
	go func() {
		management.consumeOpenvpnConnectionOutput(netConn, cmdOutputChannel, eventChannel)
		outputConsuming.Done()
	}()

	management.startMiddlewares(connection)
	defer management.stopMiddlewares(connection)

	outputConsuming.Wait()
}

func (management *Management) startMiddlewares(connection Connection) {
	for _, middleware := range management.middlewares {
		err := middleware.Start(connection)
		if err != nil {
			//TODO what we should do with errors on middleware start? Stop already running, close cmdWriter, bailout?
			//at least log errors for now
			log.Error(management.logPrefix, "Middleware startup error: ", err)
		}
	}
}

func (management *Management) stopMiddlewares(connection Connection) {
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
