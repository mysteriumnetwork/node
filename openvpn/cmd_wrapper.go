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

// NewCmdWrapper returns process wrapper for given executable
func NewCmdWrapper(executablePath, logPrefix string) *CmdWrapper {
	return &CmdWrapper{
		logPrefix:          logPrefix,
		executablePath:     executablePath,
		cmdExitError:       make(chan error, 1), //channel should have capacity to hold single process exit error
		cmdShutdownStarted: make(chan bool),
		cmdShutdownWaiter:  sync.WaitGroup{},
	}
}

// CmdWrapper struct defines process wrapper which handles clean shutdown, tracks executable exit errors, etc.
type CmdWrapper struct {
	logPrefix          string
	executablePath     string
	cmdExitError       chan error
	cmdShutdownStarted chan bool
	cmdShutdownWaiter  sync.WaitGroup
	closesOnce         sync.Once
}

// Start underlying binary defined by process wrapper with given arguments
func (cw *CmdWrapper) Start(arguments []string) (err error) {
	// Create the command
	log.Info(cw.logPrefix, "Starting cw: ", cw.executablePath, " with arguments: ", arguments)
	cmd := exec.Command(cw.executablePath, arguments...)

	// Attach monitors for stdout, stderr and exit
	cw.stdoutMonitor(cmd)
	cw.stderrMonitor(cmd)

	// Try to start the cw
	err = cmd.Start()
	if err != nil {
		return err
	}

	// Watch if the cw exits
	go cw.waitForExit(cmd)
	go cw.waitForShutdown(cmd)

	return
}

// Wait function wait until executable exits and then returns exit error reported by executable
func (cw *CmdWrapper) Wait() error {
	return <-cw.cmdExitError
}

// Stop function stops (or sends request to stop) underlying executable and waits until stdout/stderr and shutdown monitors are finished
func (cw *CmdWrapper) Stop() {
	cw.closesOnce.Do(func() {
		close(cw.cmdShutdownStarted)
	})
	cw.cmdShutdownWaiter.Wait()
}

func (cw *CmdWrapper) stdoutMonitor(cmd *exec.Cmd) {
	stdout, _ := cmd.StdoutPipe()
	go func() {
		cw.cmdShutdownWaiter.Add(1)
		defer cw.cmdShutdownWaiter.Done()

		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			log.Trace(cw.logPrefix, "Stdout: ", scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			log.Warn(cw.logPrefix, "Stdout: (failed to read: ", err, ")")
			return
		}
	}()
}

func (cw *CmdWrapper) stderrMonitor(cmd *exec.Cmd) {
	stderr, _ := cmd.StderrPipe()
	go func() {
		cw.cmdShutdownWaiter.Add(1)
		defer cw.cmdShutdownWaiter.Done()

		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			log.Warn(cw.logPrefix, "Stderr: ", scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			log.Warn(cw.logPrefix, "Stderr: (failed to read ", err, ")")
			return
		}
	}()
}

func (cw *CmdWrapper) waitForExit(cmd *exec.Cmd) {
	err := cmd.Wait()
	log.Info(cw.logPrefix, "CmdWrapper exited with error: ", err)
	cw.cmdExitError <- err
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
