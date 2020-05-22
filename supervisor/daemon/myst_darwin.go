/*
 * Copyright (C) 2020 The "MysteriumNetwork/node" Authors.
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

package daemon

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"os/user"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const pidFile = "/var/run/myst.pid"

type runOptions struct {
	uid  int
	gid  int
	home string
}

// runMyst runs mysterium node daemon. Blocks.
func (d *Daemon) runMyst(args ...string) error {
	options, err := parseRunOptions(args...)
	if err != nil {
		return err
	}
	paths := []string{
		d.cfg.OpenVPNPath,
		"/usr/bin",
		"/bin",
		"/usr/sbin",
		"/sbin",
		"/usr/local/bin",
	}

	var stdout, stderr bytes.Buffer

	cmd := exec.Cmd{
		Path: d.cfg.MystPath,
		Args: []string{
			d.cfg.MystPath,
			"--openvpn.binary", d.cfg.OpenVPNPath,
			"--mymysterium.enabled=false",
			"--ui.enable=false",
			"--usermode",
			"daemon",
		},
		Env: []string{
			"HOME=" + options.home,
			"PATH=" + strings.Join(paths, ":"),
		},
		Stdout: &stdout,
		Stderr: &stderr,
		SysProcAttr: &syscall.SysProcAttr{
			Credential: &syscall.Credential{Uid: uint32(options.uid), Gid: uint32(options.gid)},
		},
	}
	if err := cmd.Start(); err != nil {
		return err
	}

	err = runWithSuccessTimeout(cmd.Wait, 5*time.Second)
	if err != nil {
		log.Printf("myst output [err=%s]:\n%s\n%s\n", err, stderr.String(), stdout.String())
		return err
	}

	pid := cmd.Process.Pid
	if err := ioutil.WriteFile(pidFile, []byte(strconv.Itoa(pid)), 0700); err != nil {
		return err
	}

	return nil
}

func parseRunOptions(args ...string) (*runOptions, error) {
	flags := flag.NewFlagSet("", flag.ContinueOnError)
	uid := flags.Int("uid", -1, "")
	if err := flags.Parse(args[1:]); err != nil {
		return nil, fmt.Errorf("failed to parse flags: %w", err)
	}
	if *uid == -1 {
		return nil, errors.New("uid is required")
	}
	runUser, err := user.LookupId(strconv.Itoa(*uid))
	if err != nil {
		return nil, fmt.Errorf("failed to lookup user: %w", err)
	}
	gid, err := strconv.Atoi(runUser.Gid)
	if err != nil {
		return nil, fmt.Errorf("failed to parse group ID %q for user %q: %w", runUser.Gid, runUser.Username, err)
	}
	return &runOptions{
		uid:  *uid,
		gid:  gid,
		home: runUser.HomeDir,
	}, nil
}

func runWithSuccessTimeout(f func() error, timeout time.Duration) error {
	done := make(chan error, 1)
	go func() {
		done <- f()
	}()
	select {
	case <-time.After(timeout):
		return nil
	case err := <-done:
		return err
	}
}

func (d *Daemon) killMyst() error {
	if _, err := os.Stat(pidFile); os.IsNotExist(err) {
		return nil
	}
	pidFileContent, err := ioutil.ReadFile(pidFile)
	if err != nil {
		return fmt.Errorf("could not read %q: %w", pidFile, err)
	}
	pid, err := strconv.Atoi(string(pidFileContent))
	if err != nil {
		return fmt.Errorf("invalid content of %q: %w", pidFile, err)
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("could not find process %d: %w", pid, err)
	}
	if err := interrupt(proc, 3*time.Second); err != nil {
		log.Println("Attempting to kill", pid, "due to interrupt failure:", err)
		if err := kill(proc); err != nil {
			return err
		}
	} else {
		log.Println("Process successfully interrupted")
	}

	_ = os.Remove(pidFile)
	return nil
}

func interrupt(proc *os.Process, timeout time.Duration) error {
	err := proc.Signal(os.Interrupt)
	if err != nil {
		return err
	}

	interruptCh := make(chan error)
	go func() {
		state, err := proc.Wait()
		if err != nil {
			if errors.Is(err, syscall.ECHILD) {
				// no child process exists - we're good
				interruptCh <- nil
			} else {
				interruptCh <- fmt.Errorf("failed process wait: %w", err)
			}
		} else {
			if state.Exited() {
				interruptCh <- nil
			} else {
				interruptCh <- errors.New("process did not exit")
			}
		}
	}()

	select {
	case err = <-interruptCh:
	case <-time.After(timeout):
		err = errors.New("process interrupt timed out")
	}
	return err
}

func kill(proc *os.Process) error {
	err := proc.Kill()
	if err != nil {
		return fmt.Errorf("could not kill process %d: %w", proc.Pid, err)
	}
	state, err := proc.Wait()
	if err == nil && !state.Exited() {
		return fmt.Errorf("process left in running state: %d: %w", proc.Pid, err)
	}
	return nil
}
