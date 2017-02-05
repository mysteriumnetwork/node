package openvpn

import (
	"bufio"
	"bytes"
	"net"
	"net/textproto"
	"os"
	"regexp"
	"strconv"
	"sync"
	"time"

	log "github.com/cihub/seelog"
)

const MANAGEMENT_LOG_PREFIX = "[OpenVPN.managemnet] "

type Management struct {
	ManagementRead  chan string
	ManagementWrite chan string

	Path string

	events chan []string

	currentClient string
	clientEnv     map[string]string

	buffer    []byte
	waitGroup sync.WaitGroup
	shutdown  chan bool
}

func NewManagement() *Management {
	return &Management{
		ManagementRead:  make(chan string),
		ManagementWrite: make(chan string),

		events: make(chan []string),

		clientEnv: make(map[string]string, 0),
		buffer:    make([]byte, 0),
		shutdown:  make(chan bool),
	}
}

func (management *Management) Start() (path string, err error) {
	management.Path = "/tmp/openvpn-management-" + strconv.Itoa(os.Getpid()) + ".sock"
	log.Info(MANAGEMENT_LOG_PREFIX, "Connecting to socket:", management.Path)

	listener, err := net.Listen("unix", management.Path)
	if err != nil {
		log.Error(MANAGEMENT_LOG_PREFIX, err)
		return
	}

	go management.waitForShutdown(listener)
	go management.waitForConnections(listener)

	return management.Path, err
}

func (management *Management) Shutdown() {
	log.Info(MANAGEMENT_LOG_PREFIX, "Shutdown")
	close(management.shutdown)

	management.waitGroup.Wait()
	log.Info(MANAGEMENT_LOG_PREFIX, "Shutdown finished")
}

func (management *Management) waitForShutdown(listener net.Listener) {
	<-management.shutdown
	listener.Close()
}

func (management *Management) waitForConnections(listener net.Listener) {
	management.waitGroup.Add(1)
	defer management.waitGroup.Done()

	for {
		connection, err := listener.Accept()
		if err != nil {
			select {
			case <-management.shutdown:
				log.Info(MANAGEMENT_LOG_PREFIX, "Connection closed")
			default:
				log.Critical(MANAGEMENT_LOG_PREFIX, "Connection accept error:", err)
			}
			return
		}
		log.Info(MANAGEMENT_LOG_PREFIX, "Connection accepted")

		go management.server(connection)
	}
}

func (management *Management) server(connection net.Conn) {
	log.Info(MANAGEMENT_LOG_PREFIX, "Server started")

	/*
		go func() {
			//c.Write([]byte("status\n"))
			for {
				<-time.After(time.Second)
				_, err := c.Write([]byte("push \"echo hej\"\n"))
				if err != nil {
					log.Error(MANAGEMENT_LOG_PREFIX, "Failed management write:", err)
					return
				}
				log.Info(MANAGEMENT_LOG_PREFIX, "Push echo hej")
			}
		}()
	*/

	go func() {
		for {
			rows := <-management.events
			log.Info(MANAGEMENT_LOG_PREFIX, "Event received:", rows)
		}
	}()

	reader := textproto.NewReader(bufio.NewReader(connection))
	for {
		line, err := reader.ReadLine()
		if err != nil {
			return
		}
		//log.Info(MANAGEMENT_LOG_PREFIX, "Line received:", line)

		libeBytes := []byte(line)
		management.parse(libeBytes, false)
	}
}

// https://openvpn.net/index.php/open-source/documentation/miscellaneous/79-management-interface.html
func (management *Management) parse(line []byte, retry bool) {
	//log.Error(MANAGEMENT_LOG_PREFIX, "Parsing:", string(line))

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
					MANAGEMENT_LOG_PREFIX,
					"Failed to transport message (%client): %s |%s|",
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

	if len(line) > 0 && !retry {
		//log.Warn("Could not find message, adding to buffer: ", string(line))
		management.buffer = append(management.buffer, line...)
		management.buffer = append(management.buffer, '\n')
		management.parse(management.buffer, true)
	} else if len(line) > 0 {
		management.buffer = line
	}
}
