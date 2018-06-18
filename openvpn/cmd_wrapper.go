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
	"io"
	"syscall"
)

// NewCmdWrapper returns process wrapper for given executable
func NewCmdWrapper(executablePath, logPrefix string) *CmdWrapper {
	return &CmdWrapper{
		logPrefix:          logPrefix,
		executablePath:     executablePath,
		CmdExitError:       make(chan error, 1), //channel should have capacity to hold single process exit error
		cmdShutdownStarted: make(chan bool),
		cmdShutdownWaiter:  sync.WaitGroup{},
	}
}

// CmdWrapper struct defines process wrapper which handles clean shutdown, tracks executable exit errors, logs stdout and stderr to logger
type CmdWrapper struct {
	logPrefix          string
	executablePath     string
	CmdExitError       chan error
	cmdShutdownStarted chan bool
	cmdShutdownWaiter  sync.WaitGroup
	closesOnce         sync.Once
}

// Start underlying binary defined by process wrapper with given arguments
func (cw *CmdWrapper) Start(arguments []string) (err error) {
	// Create the command
	log.Info(cw.logPrefix, "Starting cmd: ", cw.executablePath, " with arguments: ", arguments)
	cmd := exec.Command(cw.executablePath, arguments...)

	// Attach logger for stdout and stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	go cw.outputToLog(stdout, "Stdout: ")
	go cw.outputToLog(stderr, "Stderr: ")

	// Try to start the cmd
	err = cmd.Start()
	if err != nil {
		return err
	}

	// Watch if the cmd exits
	go cw.waitForExit(cmd)
	go cw.waitForShutdown(cmd)

	return
}

// Wait function wait until executable exits and then returns exit error reported by executable
func (cw *CmdWrapper) Wait() error {
	return <-cw.CmdExitError
}

// Stop function stops (or sends request to stop) underlying executable and waits until stdout/stderr and shutdown monitors are finished
func (cw *CmdWrapper) Stop() {
	cw.closesOnce.Do(func() {
		close(cw.cmdShutdownStarted)
	})
	cw.cmdShutdownWaiter.Wait()
}

func (cw *CmdWrapper) outputToLog(output io.ReadCloser, streamPrefix string) {
	scanner := bufio.NewScanner(output)
	for scanner.Scan() {
		log.Trace(cw.logPrefix, streamPrefix, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		log.Warn(cw.logPrefix, streamPrefix, "(failed to read: ", err, ")")
	} else {
		log.Info(cw.logPrefix, streamPrefix, "stream ended")
	}
}

func (cw *CmdWrapper) waitForExit(cmd *exec.Cmd) {
	err := cmd.Wait()
	cw.CmdExitError <- err
}

func (cw *CmdWrapper) waitForShutdown(cmd *exec.Cmd) {
	cw.cmdShutdownWaiter.Add(1)
	defer cw.cmdShutdownWaiter.Done()

	<-cw.cmdShutdownStarted
	//First - shutdown gracefully
	//TODO - add timer and send SIGKILL after timeout?
	if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
		log.Error(cw.logPrefix, "Error killing cw = ", err)
	}
}
