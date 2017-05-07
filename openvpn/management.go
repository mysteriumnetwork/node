package openvpn

import (
	"bufio"
	"bytes"
	"net"
	"net/textproto"
	"regexp"
	"strconv"
	"sync"
	"time"

	log "github.com/cihub/seelog"
)

type Management struct {
	socketAddress string
	logPrefix     string

	events chan []string

	listenerShutdownStarted chan bool
	listenerShutdownWaiter  sync.WaitGroup
}

func NewManagement(socketAddress, logPrefix string) *Management {
	return &Management{
		socketAddress: socketAddress,
		logPrefix:     logPrefix,

		events: make(chan []string),

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
	close(management.listenerShutdownStarted)

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
				log.Critical(management.logPrefix, "Connection accept error:", err)
			}
			return
		}

		go management.serveNewConnection(connection)
	}
}

func (management *Management) serveNewConnection(connection net.Conn) {
	log.Info(management.logPrefix, "New connection started")

	go func() {
		for {
			rows := <-management.events
			log.Info(management.logPrefix, "Event received:", rows)
		}
	}()

	reader := textproto.NewReader(bufio.NewReader(connection))
	for {
		line, err := reader.ReadLine()
		if err != nil {
			log.Warn(management.logPrefix, "Connection failed to read", err)
			return
		}
		//log.Info(management.logPrefix, "Line received:", line)

		lineBytes := []byte(line)
		management.parse(lineBytes)
	}
}

// https://openvpn.net/index.php/open-source/documentation/miscellaneous/79-management-interface.html
func (management *Management) parse(line []byte) {
	//log.Info(management.logPrefix, "Parsing:", string(line))

	eventsConfig := map[string]string{
		// -- Log message output as controlled by the "log" command.
		"log": ">LOG:([^\r\n]*)$",
	}

mainLoop:
	for eventName, eventRegex := range eventsConfig {
		reg, _ := regexp.Compile(eventRegex)
		match := reg.FindAllSubmatchIndex(line, -1)
		if len(match) == 0 {
			continue
		}

		for _, row := range match {
			// Extract all strings of the current match
			strings := []string{eventName}
			for index := range row {
				if index%2 > 0 { // Skipp all odd indexes
					continue
				}

				strings = append(strings, string(line[row[index]:row[index+1]]))
			}

			// Try to deliver the message
			select {
			case management.events <- strings:
			case <-time.After(time.Second):
				log.Errorf(
					"%sFailed to transport message (%client): %s |%s|",
					management.logPrefix,
					management.events,
					eventName,
					row,
					strings,
				)
			}

			if row[0] > 0 {
				log.Warn("Trowing away message: ", strconv.Quote(string(line[:row[0]])))
			}

			// Just save the rest of the message
			line = bytes.Trim(line[row[1]:], "\x00")

			continue mainLoop
		}
	}
}
