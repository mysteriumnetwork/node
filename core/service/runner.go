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
	Wait() error
	Kill() error
}

// RunnableServiceFactory creates a new runnable service instance
type RunnableServiceFactory func() RunnableService

// Runner is responsible for starting the provided service managers
type Runner struct {
	serviceFactory  RunnableServiceFactory
	serviceManagers map[string]RunnableService
}

// NewRunner returns a new instance of runner with the runnable services
func NewRunner(factory RunnableServiceFactory) *Runner {
	return &Runner{
		serviceFactory:  factory,
		serviceManagers: make(map[string]RunnableService),
	}
}

// Register registers a service as a candidate for running
func (sr *Runner) Register(serviceType string) {
	sr.serviceManagers[serviceType] = sr.serviceFactory()
}

// StartServiceByType starts a manager of the given service type if it has one.
// It passes the options to the start method of the manager.
// On initialization failure, it returns errors.
// The error channel is used for subscribing to runtime errors.
func (sr *Runner) StartServiceByType(serviceType string, options Options, errorChannel chan error) error {
	if _, ok := sr.serviceManagers[serviceType]; !ok {
		return fmt.Errorf("unknown service type %q", serviceType)
	}

	if err := sr.serviceManagers[serviceType].Start(options); err != nil {
		return err
	}

	go func() { errorChannel <- sr.serviceManagers[serviceType].Wait() }()

	return nil
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
