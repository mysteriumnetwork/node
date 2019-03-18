/*
 * Copyright (C) 2019 The "MysteriumNetwork/node" Authors.
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

import "github.com/mysteriumnetwork/go-openvpn/openvpn"

type restartingServer struct {
	stop                chan (struct{})
	waiter              chan (error)
	openvpnFactory      func() openvpn.Process
	natPinger           NATPinger
	lastSessionShutdown chan bool
}

func (rs *restartingServer) Start() error {
	go func() {
		for {
			err := rs.natPinger.WaitForHole()
			if err != nil {
				// currently, this is never reachable, as the nat pinger has no scenarios under which it returns a non nil error
			}

			ovpn := rs.openvpnFactory()
			err = ovpn.Start()
			if err != nil {
				rs.waiter <- err
				break
			}
			waiter := make(chan error)
			go func() {
				waiter <- ovpn.Wait()
			}()
			select {
			case err = <-waiter:
				if err != nil {
					rs.waiter <- err
					return
				}
			case <-rs.stop:
				ovpn.Stop()
				rs.waiter <- nil
				return
			case <-rs.lastSessionShutdown:
				ovpn.Stop()
			}
		}
	}()
	return nil
}

func (rs *restartingServer) Wait() error {
	return <-rs.waiter
}

func (rs *restartingServer) Stop() {
	rs.stop <- struct{}{}
}
