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

package install

import (
	"errors"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/eventlog"
	"golang.org/x/sys/windows/svc/mgr"
)

const serviceName = "MysteriumVPNSupervisor"

// Install installs service for Windows. If there is previous service instance
// running it will be first uninstalled before installing new version.
func Install(options Options) error {
	if !options.valid() {
		return errInvalid
	}
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("could not connect to service manager: %w", err)
	}
	defer m.Disconnect()

	log.Info().Msg("Checking previous installation")
	if err := uninstallService(m, serviceName); err != nil {
		log.Info().Err(err).Msg("Previous service was not uninstalled")
	} else {
		if err := waitServiceDeleted(m, serviceName); err != nil {
			return fmt.Errorf("could not wait for service to deletion: %w", err)
		}
		log.Info().Msg("Uninstalled previous service")
	}

	config := mgr.Config{
		ServiceType:  windows.SERVICE_WIN32_OWN_PROCESS,
		StartType:    mgr.StartAutomatic,
		ErrorControl: mgr.ErrorNormal,
		DisplayName:  "MysteriumVPN Supervisor",
		Description:  "Handles network configuration for MysteriumVPN application.",
		Dependencies: []string{"Nsi"},
	}
	if err := installAndStartService(m, serviceName, options, config); err != nil {
		return fmt.Errorf("could not install and run service: %w", err)
	}
	return nil
}

// Uninstall uninstalls service for Windows.
func Uninstall() error {
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("could not connect to service manager: %w", err)
	}
	defer m.Disconnect()

	return uninstallService(m, serviceName)
}

func installAndStartService(m *mgr.Mgr, name string, options Options, config mgr.Config) error {
	s, err := m.CreateService(name, options.SupervisorPath, config, "-winservice")
	if err != nil {
		return fmt.Errorf("could not create service: %w", err)
	}
	defer s.Close()

	err = eventlog.InstallAsEventCreate(name, eventlog.Error|eventlog.Warning|eventlog.Info)
	if err != nil {
		s.Delete()
		return fmt.Errorf("could not configure event logging: %s", err)
	}

	if err := s.Start(); err != nil {
		return fmt.Errorf("could not start service: %w", err)
	}
	return nil
}

func uninstallService(m *mgr.Mgr, name string) error {
	s, err := m.OpenService(name)
	if err != nil {
		return fmt.Errorf("skipping uninstall, service %s is not installed", name)
	}
	defer s.Close()

	// Send stop signal and ignore errors as if service is already stopped it will
	// return error which we don't care about as we just want to delete service anyway.
	s.Control(svc.Stop)

	err = s.Delete()
	if err != nil {
		return fmt.Errorf("could not mark service for deletion: %w", err)
	}

	err = eventlog.Remove(name)
	if err != nil {
		return fmt.Errorf("cound not remove event logging: %s", err)
	}

	return nil
}

// waitServiceDeleted checks if service is deleted.
// It is considered as deleted if OpenService fails.
func waitServiceDeleted(m *mgr.Mgr, name string) error {
	timeout := time.After(10 * time.Second)
	for {
		select {
		case <-timeout:
			return errors.New("timeout waiting for service deletion")
		case <-time.After(100 * time.Millisecond):
			s, err := m.OpenService(name)
			if err != nil {
				return nil
			}
			s.Close()
		}
	}
}
