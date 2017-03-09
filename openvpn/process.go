package openvpn

import (
	"bufio"
	"os/exec"
	"sync"

	log "github.com/cihub/seelog"
)

func NewProcess(logPrefix string) *Process {
	return &Process{
		logPrefix:          logPrefix,

		cmdExitError:       make(chan error),
		cmdShutdownStarted: make(chan bool),
		cmdMonitorsWaiter:  sync.WaitGroup{},
	}
}

type Process struct {
	logPrefix          string

	cmdExitError       chan error
	cmdShutdownStarted chan bool
	cmdMonitorsWaiter  sync.WaitGroup
}

func (process *Process) Start(params ...string) (err error) {
	// Create the command
	cmd := exec.Command("openvpn", params...)

	// Attach monitors for stdout, stderr and exit
	process.stdoutMonitor(cmd)
	process.stderrMonitor(cmd)

	// Try to start the process
	err = cmd.Start()
	if err != nil {
		return err
	}

	// Watch if the process exits
	go process.waitForExit(cmd)
	go process.waitForShutdown(cmd)

	return
}

func (process *Process) Wait() {
	<-process.cmdExitError
}

func (process *Process) Stop() (err error) {
	close(process.cmdShutdownStarted)
	process.cmdMonitorsWaiter.Wait()

	return
}

func (process *Process) stdoutMonitor(cmd *exec.Cmd) {
	stdout, _ := cmd.StdoutPipe()
	go func() {
		process.cmdMonitorsWaiter.Add(1)
		defer process.cmdMonitorsWaiter.Done()

		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			log.Trace(process.logPrefix, "Stdout: ", scanner.Text())
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
		process.cmdMonitorsWaiter.Add(1)
		defer process.cmdMonitorsWaiter.Done()

		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			log.Warn(process.logPrefix, "Stderr: ", scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			log.Warn(process.logPrefix, "Stderr: (failed to read ", err, ")")
			return
		}
	}()
}

func (process *Process) waitForExit(cmd *exec.Cmd) {
	process.cmdExitError <- cmd.Wait()
}

func (process *Process) waitForShutdown(cmd *exec.Cmd) {
	process.cmdMonitorsWaiter.Add(1)
	defer process.cmdMonitorsWaiter.Done()

	select {
	// Wait for shutdown
	case <-process.cmdShutdownStarted:
		// Kill the server
		if err := cmd.Process.Kill(); err != nil {
			return
		}

		// Allow goroutine to exit
		err := <-process.cmdExitError
		log.Error(process.logPrefix, "Process killed with error = ", err)

	// Wait for exit
	case err := <-process.cmdExitError:
		log.Error(process.logPrefix, "Process cmdExitError with error = ", err)
		return
	}
}
