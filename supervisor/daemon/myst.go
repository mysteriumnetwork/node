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
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

const tequilapiHost = "http://localhost:4050"

func (d *Daemon) killMyst() error {
	err := gracefulKill(3 * time.Second)
	if err == nil {
		return nil
	}

	log.Printf("Failed to kill node gracefully, will continue with force kill: %v", err)
	pid, err := mystPid()
	if err != nil {
		return fmt.Errorf("could not get myst pid: %w", err)
	}
	if err := forceKill(pid); err != nil {
		return fmt.Errorf("could not force kill node: %w", err)
	}
	return nil
}

func gracefulKill(timeout time.Duration) error {
	client := http.Client{
		Timeout: timeout,
	}
	resp, err := client.Post(fmt.Sprintf("%s/stop", tequilapiHost), "application/json", nil)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("expected status %d, got %d", http.StatusAccepted, resp.StatusCode)
	}
	return nil
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

func mystPid() (int, error) {
	client := http.Client{
		Timeout: 3 * time.Second,
	}
	resp, err := client.Get(fmt.Sprintf("%s/healthcheck", tequilapiHost))
	if err != nil {
		return 0, err
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
