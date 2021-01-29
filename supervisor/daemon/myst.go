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
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/rs/zerolog/log"
)

const (
	tequilapiHost = "http://localhost"
	defaultPort   = 4050
)

func (d *Daemon) killMyst() error {
	log.Info().Msg("Trying to stop node process gracefully")
	err := gracefulStop(d.tequilapiURL(), 3*time.Second)
	if err == nil {
		return nil
	}

	log.Warn().Msgf("Failed to stop node gracefully, will continue with force kill: %v", err)
	pid, err := mystPid(d.tequilapiURL())
	if err != nil {
		return fmt.Errorf("could not get myst pid: %w", err)
	}
	if err := forceKill(pid); err != nil {
		return fmt.Errorf("could not force kill node: %w", err)
	}
	return nil
}

func (d *Daemon) setTequilapiPort(cmd []string) error {
	if len(cmd) < 2 {
		return fmt.Errorf("expected 2 arguments")
	}
	port, err := strconv.ParseUint(cmd[1], 10, 16)
	if err != nil {
		return err
	}
	log.Info().Msgf("Changing tequilapi port to: %d", port)
	d.tequilapiPort = uint16(port)
	return nil
}

func (d *Daemon) tequilapiURL() string {
	return fmt.Sprintf("%s:%d", tequilapiHost, d.tequilapiPort)
}

func gracefulStop(tequilapiURL string, timeout time.Duration) error {
	client := http.Client{Timeout: timeout}
	resp, err := client.Post(fmt.Sprintf("%s/stop", tequilapiURL), "application/json", nil)
	if err != nil {
		return fmt.Errorf("could not call stop: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("expected status %d, got %d", http.StatusAccepted, resp.StatusCode)
	}

	timeoutCh := time.After(timeout)
	for {
		select {
		case <-timeoutCh:
			return errors.New("timeout waiting for myst to exit")
		default:
			_, err := mystPid(tequilapiURL)
			if err != nil {
				return nil
			}
			time.Sleep(500 * time.Millisecond)
		}
	}
}

func forceKill(pid int) error {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("could not find process %d: %w", pid, err)
	}
	return kill(proc)
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

func mystPid(tequilapiURL string) (int, error) {
	client := http.Client{
		Timeout: 3 * time.Second,
	}
	resp, err := client.Get(fmt.Sprintf("%s/healthcheck", tequilapiURL))
	if err != nil {
		return 0, fmt.Errorf("could not call healthcheck: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	type healthcheck struct {
		Process int `json:"process"`
	}
	hz := healthcheck{}
	if err := json.NewDecoder(resp.Body).Decode(&hz); err != nil {
		return 0, fmt.Errorf("could not parse health check JSON: %w", err)
	}
	return hz.Process, nil
}
