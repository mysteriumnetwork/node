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
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestJob(t *testing.T) {
	var ops uint64
	var actualWork = func() string {
		atomic.AddUint64(&ops, 1)
		return "doing actual work"
	}
	f := func(stop <-chan struct{}) {
		for {
			actualWork()
			select {
			case <-stop:
				return
			case <-time.After(time.Millisecond):
			}
		}
	}
	j := newJob(f, time.Millisecond*5)
	j.Start()

	go func() {
		time.Sleep(time.Millisecond * 3)
		j.Stop()
	}()

	<-j.Done()

	loaded := atomic.LoadUint64(&ops)
	// assert work was done
	assert.Greater(t, loaded, uint64(0))

	nj := j.Restart()
	<-nj.Done()

	// assert more work was done
	current := atomic.LoadUint64(&ops)
	assert.Greater(t, current, loaded)
}
