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
	"fmt"

	"github.com/rs/zerolog/log"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/eventlog"
	"golang.org/x/sys/windows/svc/mgr"
)

const serviceName = "MystSupervisor"

// Install installs service for Windows.
func Install(options Options) error {
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("could not connect to service manager: %w", err)
	}
	defer m.Disconnect()

	log.Info().Msg("Cleaning up previous installation")
	uninstallService(m, serviceName)

	config := mgr.Config{
		ServiceType:  windows.SERVICE_WIN32_OWN_PROCESS,
		StartType:    mgr.StartAutomatic,
		ErrorControl: mgr.ErrorNormal,
		DisplayName:  "MystSupervisor Service",
		Description:  "Mysterium Network dApp supervisor service is responsible for managing network configurations",
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
	s, err := m.CreateService(name, options.SupervisorPath, config, "")
	if err != nil {
		return err
	}
	defer s.Close()

	err = eventlog.InstallAsEventCreate(name, eventlog.Error|eventlog.Warning|eventlog.Info)
	if err != nil {
		s.Delete()
		return fmt.Errorf("SetupEventLogSource() failed: %s", err)
	}

	if err := s.Start(); err != nil {
		return fmt.Errorf("could not start service: %w", err)
	}
	return nil
}

func uninstallService(m *mgr.Mgr, name string) error {
	s, err := m.OpenService(name)
	if err != nil {
		return fmt.Errorf("service %s is not installed", name)
	}
	defer s.Close()

	s.Control(svc.Stop)

	err = s.Delete()
	if err != nil {
		return fmt.Errorf("could not mark service for deletion: %w", err)
	}

	err = eventlog.Remove(name)
	if err != nil {
		return fmt.Errorf("RemoveEventLogSource() failed: %s", err)
	}
	return nil
}
