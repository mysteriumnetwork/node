/*
 * Copyright (C) 2018 The "MysteriumNetwork/node" Authors.
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

package service

import (
	"fmt"
)

// RunnableService represents a runnable service
type RunnableService interface {
	Start(options Options) (err error)
	Kill() error
}

// Runner is responsible for starting the provided service managers
type Runner struct {
	serviceManagers map[string]RunnableService
}

// NewRunner returns a new instance of runner with the runnable services
func NewRunner() *Runner {
	return &Runner{
		serviceManagers: make(map[string]RunnableService),
	}
}

// Register registers a service as a candidate for running
func (sr *Runner) Register(serviceType string, service RunnableService) {
	sr.serviceManagers[serviceType] = service
}

// StartServiceByType starts a manager of the given service type if it has one. The method blocks.
// It passes the options to the start method of the manager.
// If an error occurs in the underlying service, the error is then returned.
func (sr *Runner) StartServiceByType(serviceType string, options Options) error {
	if _, ok := sr.serviceManagers[serviceType]; !ok {
		return fmt.Errorf("unknown service type %q", serviceType)
	}

	return sr.serviceManagers[serviceType].Start(options)
}

// KillAll kills all service managers
func (sr *Runner) KillAll() []error {
	errors := make([]error, 0)
	for _, serviceManager := range sr.serviceManagers {
		if err := serviceManager.Kill(); err != nil {
			errors = append(errors, err)
		}
	}
	return errors
}
