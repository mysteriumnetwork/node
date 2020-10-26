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

package discovery

import (
	"fmt"
)

// Worker continuously runs discovery process in node's background.
type Worker interface {
	Start() error
	Stop()
}

type workerComposite []Worker

// NewWorker creates an instance of composite worker.
func NewWorker(workers ...Worker) *workerComposite {
	wc := workerComposite(workers)
	return &wc
}

// AddWorker adds worker to set of workers.
func (wc *workerComposite) AddWorker(worker Worker) {
	*wc = append(*wc, worker)
}

// Start starts all workers.
func (wc *workerComposite) Start() error {
	for _, worker := range *wc {
		if err := worker.Start(); err != nil {
			return fmt.Errorf("failed to start worker: %w", err)
		}
	}

	return nil
}

// Start starts all workers.
func (wc *workerComposite) Stop() {
	for _, worker := range *wc {
		worker.Stop()
	}
}
