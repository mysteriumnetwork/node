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

type job struct {
	f       func(stop <-chan struct{})
	jobStop chan struct{}
	timeout time.Duration
	once    sync.Once

	lock sync.Mutex
}

func newJob(toRun func(stop <-chan struct{}), timeout time.Duration) *job {
	j := &job{
		f:       toRun,
		timeout: timeout,
	}

	return j.init()
}

func (j *job) init() *job {
	j.once = sync.Once{}
	j.jobStop = make(chan struct{})
	return j
}

func (j *job) Stop() *job {
	j.lock.Lock()
	defer j.lock.Unlock()

	return j.stop()
}

func (j *job) stop() *job {
	j.once.Do(func() {
		close(j.jobStop)
	})
	return j
}

func (j *job) Restart() *job {
	j.lock.Lock()
	defer j.lock.Unlock()
	j.stop()
	new := newJob(j.f, j.timeout)
	new = new.start()
	return new
}

func (j *job) Start() *job {
	j.lock.Lock()
	defer j.lock.Unlock()
	return j.start()
}

func (j *job) Done() <-chan struct{} {
	return j.jobStop
}

func (j *job) start() *job {
	go func() {
		select {
		case <-time.After(j.timeout):
			j.Stop()
			return
		case <-j.jobStop:
			return
		}
	}()
	go j.f(j.jobStop)
	return j
}
