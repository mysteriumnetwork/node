/*
 * Copyright (C) 2021 The "MysteriumNetwork/node" Authors.
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

package pingpong

import (
	"sync"
	"time"
)

type balanceSyncer struct {
	jobs map[string]*job
	lock sync.Mutex
}

func newBalanceSyncer() *balanceSyncer {
	return &balanceSyncer{
		jobs: make(map[string]*job),
	}
}

// PeriodiclySyncBalance takes in the given job key, check if the job is already running. If so, it restarts the job to extend its lifetime. Otherwise it creates a new job and starts it.
func (bs *balanceSyncer) PeriodiclySyncBalance(jobKey string, toRun func(stop <-chan struct{}), timeout time.Duration) (*job, bool) {
	bs.lock.Lock()
	defer bs.lock.Unlock()

	if v, ok := bs.jobs[jobKey]; ok {
		v = v.Restart()
		bs.jobs[jobKey] = v
		return v, true
	}

	j := newJob(toRun, timeout).Start()
	bs.jobs[jobKey] = j
	return j, false
}
