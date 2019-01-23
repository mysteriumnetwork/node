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
	"github.com/mysteriumnetwork/node/utils"
)

// RunnableService represents a runnable service
type RunnableService interface {
	Stop() error
}

// Pool is responsible for supervising running services
type Pool struct {
	services []RunnableService
}

// NewPool returns a empty service pool
func NewPool() *Pool {
	return &Pool{}
}

// Add registers a service to running services pool
func (sr *Pool) Add(service RunnableService) {
	sr.services = append(sr.services, service)
}

// StopAll kills all running services
func (sr *Pool) StopAll() error {
	errStop := utils.ErrorCollection{}
	for _, service := range sr.services {
		if err := service.Stop(); err != nil {
			errStop.Add(err)
		}
	}

	return errStop.Errorf("Some services did not stop: %v", ". ")
}
