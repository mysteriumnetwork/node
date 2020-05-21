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

package wginterface

import (
	"fmt"
	"log"

	"golang.zx2c4.com/wireguard/device"
	"golang.zx2c4.com/wireguard/ipc"
	"golang.zx2c4.com/wireguard/tun"
)

// New creates new WgInterface instance.
func New(interfaceName string, uid string) (*WgInterface, error) {
	log.Println("Creating Wintun interface")
	// guid := &windows.GUID{}
	wintun, err := tun.CreateTUN(interfaceName, 0)
	if err != nil {
		return nil, fmt.Errorf("could not create wintun: %w", err)
	}
	nativeTun := wintun.(*tun.NativeTun)
	wintunVersion, ndisVersion, err := nativeTun.Version()
	if err != nil {
		log.Printf("Warning: unable to determine Wintun version: %v", err)
	} else {
		log.Printf("Using Wintun/%s (NDIS %s)", wintunVersion, ndisVersion)
	}

	log.Println("Creating interface instance")
	// TODO: User ring logger?
	logger := device.NewLogger(device.LogLevelDebug, fmt.Sprintf("(%s) ", interfaceName))
	logger.Info.Println("Starting wireguard-go version", device.WireGuardGoVersion)
	wgDevice := device.NewDevice(wintun, logger)

	log.Println("Setting interface configuration")
	uapi, err := ipc.UAPIListen(interfaceName)
	if err != nil {
		return nil, fmt.Errorf("could not listen for user API wg configuration: %w", err)
	}

	wg := &WgInterface{
		Name:   interfaceName,
		device: wgDevice,
		uapi:   uapi,
	}
	go wg.handleUAPI()

	return wg, nil
}

// handleUAPI listens for WireGuard configuration changes via user space socket.
func (a *WgInterface) handleUAPI() {
	for {
		conn, err := a.uapi.Accept()
		if err != nil {
			log.Println("Closing UAPI listener, err:", err)
			return
		}
		go a.device.IpcHandle(conn)
	}
}

// Down closes device and user space api socket.
func (a *WgInterface) Down() {
	if err := a.uapi.Close(); err != nil {
		log.Printf("could not close uapi socket: %w", err)
	}
	a.device.Close()
}
