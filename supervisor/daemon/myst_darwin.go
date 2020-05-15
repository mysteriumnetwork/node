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
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const pidFile = "/var/run/myst.pid"

// runMyst runs mysterium node daemon. Blocks.
func (d *Daemon) runMyst(args ...string) error {
	flags := flag.NewFlagSet("", flag.ContinueOnError)
	uid := flags.Int("uid", 0, "")
	gid := flags.Int("gid", 0, "")
	if err := flags.Parse(args[1:]); err != nil {
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
	sysProcAttr := &syscall.SysProcAttr{}
	sysProcAttr.Credential = &syscall.Credential{Uid: uint32(*uid), Gid: uint32(*gid)}

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
			"HOME=" + d.cfg.MystHome,
			"PATH=" + strings.Join(paths, ":"),
		},
		Stdout:      &stdout,
		Stderr:      &stderr,
		SysProcAttr: sysProcAttr,
	}
	if err := cmd.Start(); err != nil {
		return err
	}

	err := runWithSuccessTimeout(cmd.Wait, 5*time.Second)
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
	if err := proc.Signal(syscall.SIGINT); err != nil {
		return fmt.Errorf("could not interrupt process %d: %w", pid, err)
	}
	// TODO kill if doesn't terminate in 5secs
	_ = os.Remove(pidFile)
	return nil
}
