/*
 * Copyright (C) 2024 The "MysteriumNetwork/node" Authors.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the free Software Foundation, either version 3 of the License, or
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
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/mysteriumnetwork/node/pb"
	"github.com/mysteriumnetwork/node/trace"
)

func TestSession_DataRace(t *testing.T) {

	service := &Instance{
		location: mockLocationResolver{},
	}

	active := new(atomic.Bool)
	active.Store(true)

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()

		for i := 0; i < 100; i++ {
			session, _ := NewSession(service, &pb.SessionRequest{}, trace.NewTracer(""))
			_ = session

			time.Sleep(time.Millisecond)
		}
		active.Store(false)
	}()

	go func() {
		defer wg.Done()

		for active.Load() == true {
			service.proposalWithCurrentLocation()

			time.Sleep(time.Millisecond)
		}
	}()
	wg.Wait()

}
