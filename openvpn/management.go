package openvpn

import (
	"bufio"
	"net"
	"net/textproto"
	"sync"
	"time"

	log "github.com/cihub/seelog"
)

// https://openvpn.net/index.php/open-source/documentation/miscellaneous/79-management-interface.html
type Management struct {
	socketAddress string
	logPrefix     string

	lineReceiver chan string
	middlewares  []ManagementMiddleware

	listenerShutdownStarted chan bool
	listenerShutdownWaiter  sync.WaitGroup
}

type ManagementMiddleware interface {
	Start(connection net.Conn) error
	Stop() error
	ConsumeLine(line string) (consumed bool, err error)
}

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
	close(management.listenerShutdownStarted)

	management.listenerShutdownWaiter.Wait()
	log.Info(management.logPrefix, "Shutdown finished")
}

func (management *Management) waitForShutdown(listener net.Listener) {
	<-management.listenerShutdownStarted

	for _, middleware := range management.middlewares {
		middleware.Stop()
	}

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
		middleware.Start(connection)
	}

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

func (management *Management) deliverLines() {
	for {
		line := <-management.lineReceiver
		log.Debug(management.logPrefix, "Line delivering: ", line)

		lineConsumed := false
		for _, middleware := range management.middlewares {
			consumed, err := middleware.ConsumeLine(line)
			if err != nil {
				log.Error(management.logPrefix, "Failed to deliver line: ", line, ". ", err)
			}

			lineConsumed = lineConsumed || consumed
		}
		if !lineConsumed {
			log.Warn(management.logPrefix, "Line not delivered: ", line)
		}
	}
}
