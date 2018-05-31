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

package openvpn

import (
	"bufio"
	"os/exec"
	"sync"

	log "github.com/cihub/seelog"
	"syscall"
)

// NewProcess returns process wrapper for given executable
func NewProcess(openvpnBinary, logPrefix string) *Process {
	return &Process{
		logPrefix:          logPrefix,
		openvpnBinary:      openvpnBinary,
		cmdExitError:       make(chan error, 1), //channel should have capacity to hold single process exit error
		cmdShutdownStarted: make(chan bool),
		cmdShutdownWaiter:  sync.WaitGroup{},
	}
}

// Process struct defines process wrapper which handles clean shutdown, tracks executable exit errors, etc.
type Process struct {
	logPrefix          string
	openvpnBinary      string
	cmdExitError       chan error
	cmdShutdownStarted chan bool
	cmdShutdownWaiter  sync.WaitGroup
	closesOnce         sync.Once
}

// Start underlying binary defined by process wrapper with given arguments
func (process *Process) Start(arguments []string) (err error) {
	// Create the command
	log.Info(process.logPrefix, "Starting process with arguments: ", arguments)
	cmd := exec.Command(process.openvpnBinary, arguments...)

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

// Wait function wait until executable exits and then returns exit error reported by executable
func (process *Process) Wait() error {
	return <-process.cmdExitError
}

// Stop function stops (or sends request to stop) underlying executable and waits until stdout/stderr and shutdown monitors are finished
func (process *Process) Stop() {
	process.closesOnce.Do(func() {
		close(process.cmdShutdownStarted)
	})
	process.cmdShutdownWaiter.Wait()
}

func (process *Process) stdoutMonitor(cmd *exec.Cmd) {
	stdout, _ := cmd.StdoutPipe()
	go func() {
		process.cmdShutdownWaiter.Add(1)
		defer process.cmdShutdownWaiter.Done()

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
		process.cmdShutdownWaiter.Add(1)
		defer process.cmdShutdownWaiter.Done()

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
	err := cmd.Wait()
	log.Info(process.logPrefix, "Process exited with error: ", err)
	process.cmdExitError <- err
}

func (process *Process) waitForShutdown(cmd *exec.Cmd) {
	process.cmdShutdownWaiter.Add(1)
	defer process.cmdShutdownWaiter.Done()

	<-process.cmdShutdownStarted
	//First - shutdown gracefully
	//TODO - add timer and send SIGKILL after timeout?
	if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
		log.Error(process.logPrefix, "Error killing process = ", err)
	}
}
