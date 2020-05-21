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
	"log"

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
		return err
	}
	defer m.Disconnect()

	config := mgr.Config{
		ServiceType:  windows.SERVICE_WIN32_OWN_PROCESS,
		StartType:    mgr.StartAutomatic,
		ErrorControl: mgr.ErrorNormal,
		DisplayName:  "MystSupervisor Service",
		Description:  "Mysterium Network dApp supervisor service is responsible for manages network configurations",
	}

	if err := uninstallService(m, serviceName); err != nil {
		log.Printf("Failed to remove service: %v", err)
	}

	if err := installAndStartService(m, serviceName, options, config); err != nil {
		return fmt.Errorf("could not install and run service: %w", err)
	}
	return nil
}

func installAndStartService(m *mgr.Mgr, name string, options Options, config mgr.Config) error {
	s, err := m.OpenService(name)
	if err == nil {
		s.Close()
		return fmt.Errorf("service %s already exists", name)
	}

	s, err = m.CreateService(name, options.SupervisorPath, config, "")
	if err != nil {
		return err
	}
	defer s.Close()
	err = eventlog.Remove(name)
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
	log.Println("Detected previously installed service, uninstalling...")

	s.Control(svc.Stop)
	err = s.Delete()
	err2 := s.Close()
	if err != nil {
		return fmt.Errorf("could not delete service: %w", err)
	}
	if err2 != nil {
		return fmt.Errorf("could not close service handle: %w", err)
	}
	err = eventlog.Remove(name)
	if err != nil {
		return fmt.Errorf("RemoveEventLogSource() failed: %s", err)
	}
	return nil
}
