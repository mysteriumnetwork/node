package management

import (
	"bufio"
	"net"
	"net/textproto"
	"sync"
	"time"

	log "github.com/cihub/seelog"
)

// Management structure represents connection and interface to openvpn management
type Management struct {
	socketAddress string
	logPrefix     string

	lineReceiver chan string
	middlewares  []ManagementMiddleware

	listenerShutdownStarted chan bool
	listenerShutdownWaiter  sync.WaitGroup
	closesOnce              sync.Once
}

// NewManagement creates new manager for given sock address, uses given log prefix for logging and takes a list of middlewares
func NewManagement(socketAddress, logPrefix string, middlewares ...ManagementMiddleware) *Management {
	return &Management{
		socketAddress: socketAddress,
		logPrefix:     logPrefix,

		lineReceiver: make(chan string),
		middlewares:  middlewares,

		listenerShutdownStarted: make(chan bool),
		listenerShutdownWaiter:  sync.WaitGroup{},
	}
}

// SocketAddress returns management socket address
func (management *Management) SocketAddress() string {
	return management.socketAddress
}

func (management *Management) Start() error {
	log.Info(management.logPrefix, "Connecting to socket:", management.socketAddress)

	listener, err := net.Listen("unix", management.socketAddress)
	if err != nil {
		log.Error(management.logPrefix, err)
		return err
	}

	go management.waitForShutdown(listener)
	go management.deliverLines()
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

func (management *Management) serveNewConnection(connection net.Conn) {
	log.Info(management.logPrefix, "New connection started")

	for _, middleware := range management.middlewares {
		err := middleware.Start(textproto.NewWriter(bufio.NewWriter(connection)))
		if err != nil {
			//TODO what we should do with errors on middleware start? Stop already running, close conn, bailout?
			//at least log errors for now
			log.Error(management.logPrefix, "Middleware startup error: ", err)
		}
	}

	defer management.cleanConnection(connection)

	reader := textproto.NewReader(bufio.NewReader(connection))
	for {
		line, err := reader.ReadLine()
		if err != nil {
			log.Warn(management.logPrefix, "Connection failed to read. ", err)
			return
		}
		log.Debug(management.logPrefix, "Line received: ", line)

		// Try to deliver the message
		select {
		case management.lineReceiver <- line:
		case <-time.After(time.Second):
			log.Error(management.logPrefix, "Failed to transport line: ", line)
		}
	}
}

func (management *Management) cleanConnection(connection net.Conn) {
	for _, middleware := range management.middlewares {
		err := middleware.Stop(textproto.NewWriter(bufio.NewWriter(connection)))
		if err != nil {
			//log error but do not stop cleaning process
			log.Warn(management.logPrefix, "Middleware stop error. ", err)
		}
	}
	connection.Close()
}

func (management *Management) deliverLines() {
	for {
		line := <-management.lineReceiver
		log.Trace(management.logPrefix, "Line delivering: ", line)

		lineConsumed := false
		for _, middleware := range management.middlewares {
			consumed, err := middleware.ConsumeLine(line)
			if err != nil {
				log.Error(management.logPrefix, "Failed to deliver line: ", line, ". ", err)
			}

			lineConsumed = lineConsumed || consumed
		}
		if !lineConsumed {
			log.Trace(management.logPrefix, "Line not delivered: ", line)
		}
	}
}
