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

import (
	log "github.com/cihub/seelog"
	"github.com/mysteriumnetwork/go-openvpn/openvpn"
)

const restartingServerLogPrefix = "[nat-openvpn] "

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
			waiterProcess := make(chan error)
			holeWaiter := make(chan error)
			log.Info(restartingServerLogPrefix, "constructing openvpn")
			ovpn := rs.openvpnFactory()

			go func() {
				log.Info(restartingServerLogPrefix, "waiting for nat hole")
				holeWaiter <- rs.natPinger.WaitForHole()
			}()

			// wait for the hole to be punched, or the stop to be sent
			select {
			case <-holeWaiter:
				log.Info(restartingServerLogPrefix, "starting openvpn")
				err := ovpn.Start()
				if err != nil {
					rs.waiter <- err
					return
				}
				go func() {
					log.Info(restartingServerLogPrefix, "waiting for openvpn")
					waiterProcess <- ovpn.Wait()
				}()
			case <-rs.stop:
				rs.stopCleanup(ovpn)
				return
			}

			// wait for stops, last session events or openvpn process exits
			select {
			case err := <-waiterProcess:
				log.Info(restartingServerLogPrefix, "waiter err", err)
				if err != nil {
					rs.waiter <- err
					return
				}
			case <-rs.stop:
				rs.stopCleanup(ovpn)
				return
			case <-rs.lastSessionShutdown:
				log.Info(restartingServerLogPrefix, "last session called")
				ovpn.Stop()
				log.Info(restartingServerLogPrefix, "last session called -> returning")
			}
		}
	}()
	return nil
}

func (rs *restartingServer) stopCleanup(openvpn openvpn.Process) {
	log.Info(restartingServerLogPrefix, "stop called")
	openvpn.Stop()
	rs.waiter <- nil
	log.Info(restartingServerLogPrefix, "stopped -> returning")
}

func (rs *restartingServer) Wait() error {
	return <-rs.waiter
}

func (rs *restartingServer) Stop() {
	close(rs.stop)
}
