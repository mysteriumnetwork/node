package openvpn

import (
	"bufio"
	"os/exec"
	"sync"

	log "github.com/cihub/seelog"
)

func NewClient(config *ClientConfig) *Client {
	return &Client{
		config:     config,
		management: NewManagement(),
		shutdown: make(chan bool),
	}
}

type Client struct {
	config     *ClientConfig
	management *Management

	StdOut     chan string
	StdErr     chan string

	shutdown  chan bool
	waitGroup sync.WaitGroup
}

func (client *Client) Start() (err error) {
	// Start the management interface (if it isnt already started)
	path, err := client.management.Start()
	if err != nil {
		return err
	}

	// Add the management interface path to the config
	client.config.SetManagementPath(path)

	return client.establishConnection()
}

func (client *Client) Stop() (err error) {
	close(client.shutdown)
	client.waitGroup.Wait()

	return
}

func (client *Client) Shutdown() (err error) {
	client.Stop()
	client.management.Shutdown()

	return
}

func (client *Client) establishConnection() (err error) {
	// Fetch the current params
	params, err := client.config.Validate()
	log.Info("Validating params:", params)
	if err != nil {
		return err
	}

	// Create the command
	cmd := exec.Command("openvpn", params...)

	// Attatch monitors for stdout, stderr and exit
	release := make(chan bool)
	defer close(release)
	client.ProcessMonitor(cmd, release)

	// Try to start the process
	err = cmd.Start()
	if err != nil {
		return err
	}

	return
}

func (client *Client) ProcessMonitor(cmd *exec.Cmd, release chan bool) {
	client.stdoutMonitor(cmd)
	client.stderrMonitor(cmd)

	go func() {
		client.waitGroup.Add(1)
		defer client.waitGroup.Done()

		// Watch if the process exits
		done := make(chan error)
		go func() {
			<-release // Wait for the process to start
			done <- cmd.Wait()
		}()

		// Wait for shutdown or exit
		select {
		case <-client.shutdown:
			// Kill the server
			if err := cmd.Process.Kill(); err != nil {
				return
			}
			err := <-done // allow goroutine to exit
			log.Error("process killed with error = ", err)
		case err := <-done:
			log.Error("process done with error = ", err)
			return
		}

	}()
}

func (client *Client) stdoutMonitor(cmd *exec.Cmd) {
	stdout, _ := cmd.StdoutPipe()
	go func() {
		client.waitGroup.Add(1)
		defer client.waitGroup.Done()

		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			select {
			case client.StdOut <- scanner.Text():
			default:
				log.Trace("OPENVPN stdout: ", scanner.Text())
			}

		}
		if err := scanner.Err(); err != nil {
			log.Warn("OPENVPN stdout: (failed to read: ", err, ")")
			return
		}
	}()
}

func (client *Client) stderrMonitor(cmd *exec.Cmd) {
	stderr, _ := cmd.StderrPipe()
	go func() {
		client.waitGroup.Add(1)
		defer client.waitGroup.Done()

		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			select {
			case client.StdErr <- scanner.Text():
			default:
				log.Warn("OPENVPN stderr: ", scanner.Text())
			}
		}
		if err := scanner.Err(); err != nil {
			log.Warn("OPENVPN stderr: (failed to read ", err, ")")
			return
		}
	}()
}
