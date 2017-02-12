package openvpn

import (
	"bufio"
	"os/exec"
	"sync"

	log "github.com/cihub/seelog"
)

func NewProcess(logPrefix string) *Process {
	return &Process{
		logPrefix: logPrefix,
		shutdown: make(chan bool),
	}
}

type Process struct {
	logPrefix string

	StdOut     chan string
	StdErr     chan string

	shutdown  chan bool
	waitGroup sync.WaitGroup
}

func (process *Process) Start(params ...string) (err error) {
	// Create the command
	cmd := exec.Command("openvpn", params...)

	// Attach monitors for stdout, stderr and exit
	release := make(chan bool)
	defer close(release)
	process.ProcessMonitor(cmd, release)

	// Try to start the process
	err = cmd.Start()
	if err != nil {
		return err
	}

	return
}

func (process *Process) Stop() (err error) {
	close(process.shutdown)
	process.waitGroup.Wait()

	return
}

func (process *Process) ProcessMonitor(cmd *exec.Cmd, release chan bool) {
	process.stdoutMonitor(cmd)
	process.stderrMonitor(cmd)

	go func() {
		process.waitGroup.Add(1)
		defer process.waitGroup.Done()

		// Watch if the process exits
		done := make(chan error)
		go func() {
			<-release // Wait for the process to start
			done <- cmd.Wait()
		}()

		// Wait for shutdown or exit
		select {
		case <-process.shutdown:
			// Kill the server
			if err := cmd.Process.Kill(); err != nil {
				return
			}
			err := <-done // allow goroutine to exit
			log.Error(process.logPrefix, "Process killed with error = ", err)
		case err := <-done:
			log.Error(process.logPrefix, "Process done with error = ", err)
			return
		}

	}()
}

func (process *Process) stdoutMonitor(cmd *exec.Cmd) {
	stdout, _ := cmd.StdoutPipe()
	go func() {
		process.waitGroup.Add(1)
		defer process.waitGroup.Done()

		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			select {
			case process.StdOut <- scanner.Text():
			default:
				log.Trace(process.logPrefix, "Stdout: ", scanner.Text())
			}

		}
		if err := scanner.Err(); err != nil {
			log.Warn(process.logPrefix, "Stdout: (failed to read: ", err, ")")
			return
		}
	}()
}

func (process *Process) stderrMonitor(cmd *exec.Cmd) {
	stderr, _ := cmd.StderrPipe()
	go func() {
		process.waitGroup.Add(1)
		defer process.waitGroup.Done()

		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			select {
			case process.StdErr <- scanner.Text():
			default:
				log.Warn(process.logPrefix, "Stderr: ", scanner.Text())
			}
		}
		if err := scanner.Err(); err != nil {
			log.Warn(process.logPrefix, "Stderr: (failed to read ", err, ")")
			return
		}
	}()
}
