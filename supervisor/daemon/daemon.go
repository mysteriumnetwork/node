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

package daemon

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"strings"

	"github.com/rs/zerolog/log"

	"github.com/mysteriumnetwork/node/metadata"
	"github.com/mysteriumnetwork/node/router/network"
	"github.com/mysteriumnetwork/node/services/wireguard/wgcfg"
	"github.com/mysteriumnetwork/node/supervisor/daemon/transport"
	"github.com/mysteriumnetwork/node/supervisor/daemon/wireguard"
)

// Daemon - supervisor process.
type Daemon struct {
	monitor       *wireguard.Monitor
	tequilapiPort uint16
}

// New creates a new daemon.
func New() Daemon {
	return Daemon{monitor: wireguard.NewMonitor(), tequilapiPort: defaultPort}
}

// Start supervisor daemon. Blocks.
func (d *Daemon) Start(options transport.Options) error {
	return transport.Start(d.dialog, options)
}

// dialog talks to the client via established connection.
func (d *Daemon) dialog(conn io.ReadWriter) {
	scan := bufio.NewScanner(conn)
	answer := responder{conn}
	for scan.Scan() {
		line := scan.Bytes()
		log.Debug().Msgf("> %s", line)
		cmd := strings.Split(string(line), " ")
		op := strings.ToLower(cmd[0])
		switch op {
		case commandVersion:
			answer.ok(metadata.VersionAsString())
		case commandBye:
			answer.ok("bye")
			return
		case commandPing:
			answer.ok("pong")
		case commandWgUp:
			up, err := d.wgUp(cmd...)
			if err != nil {
				log.Err(err).Msgf("%s failed", commandWgUp)
				answer.err(err)
			} else {
				answer.ok(up)
			}
		case commandWgDown:
			err := d.wgDown(cmd...)
			if err != nil {
				log.Err(err).Msgf("%s failed", commandWgDown)
				answer.err(err)
			} else {
				answer.ok()
			}
		case commandWgStats:
			stats, err := d.wgStats(cmd...)
			if err != nil {
				log.Err(err).Msgf("%s failed", commandWgStats)
				answer.err(err)
			} else {
				answer.ok(stats)
			}
		case commandKill:
			if err := d.killMyst(); err != nil {
				log.Err(err).Msgf("%s failed", commandKill)
				answer.err(err)
			} else {
				answer.ok()
			}
		case commandTequilapiSetPort:
			if err := d.setTequilapiPort(cmd); err != nil {
				log.Err(err).Msgf("%s failed", commandTequilapiSetPort)
				answer.err(err)
			} else {
				answer.ok()
			}
		case commandDiscoverGateway:
			t := &network.RoutingTable{}
			gw, err := t.DiscoverGateway()
			if err != nil {
				log.Err(err).Msgf("%s failed", commandDiscoverGateway)
				answer.err(err)
			} else {
				answer.ok(gw.String())
			}
		case commandExcludeRoute:
			if err := d.excludeRoute(cmd...); err != nil {
				log.Err(err).Msgf("%s failed", commandDiscoverGateway)
				answer.err(err)
			} else {
				answer.ok()
			}
		case commandDeleteRoute:
			if err := d.deleteRoute(cmd...); err != nil {
				log.Err(err).Msgf("%s failed", commandDiscoverGateway)
				answer.err(err)
			} else {
				answer.ok()
			}
		}
	}
}

func (d *Daemon) excludeRoute(args ...string) error {
	flags := flag.NewFlagSet("", flag.ContinueOnError)

	ip := flags.String("ip", "", "Destination IP address")
	gw := flags.String("gw", "", "Gateway")

	if err := flags.Parse(args[1:]); err != nil {
		return err
	}

	if *ip == "" {
		return errors.New("-ip is required")
	}
	if *gw == "" {
		return errors.New("-gw is required")
	}

	ipAddr := net.ParseIP(*ip)
	gwAddr := net.ParseIP(*gw)

	t := &network.RoutingTable{}
	return t.ExcludeRule(ipAddr, gwAddr)
}

func (d *Daemon) deleteRoute(args ...string) error {
	flags := flag.NewFlagSet("", flag.ContinueOnError)

	ip := flags.String("ip", "", "Destination IP address")
	gw := flags.String("gw", "", "Gateway")

	if err := flags.Parse(args[1:]); err != nil {
		return err
	}

	if *ip == "" {
		return errors.New("-ip is required")
	}
	if *gw == "" {
		return errors.New("-gw is required")
	}

	ipAddr := net.ParseIP(*ip)
	gwAddr := net.ParseIP(*gw)

	t := &network.RoutingTable{}
	return t.DeleteRule(ipAddr, gwAddr)
}

func (d *Daemon) wgUp(args ...string) (interfaceName string, err error) {
	flags := flag.NewFlagSet("", flag.ContinueOnError)
	deviceConfigStr := flags.String("config", "", "Device configuration JSON string")
	uid := flags.String("uid", "", "User ID."+
		" On POSIX systems, this is a decimal number representing the uid."+
		" On Windows, this is a security identifier (SID) in a string format.")
	if err := flags.Parse(args[1:]); err != nil {
		return "", err
	}
	if *deviceConfigStr == "" {
		return "", errors.New("-config is required")
	}
	if *uid == "" {
		return "", errors.New("-uid is required")
	}

	configJSON, err := base64.StdEncoding.DecodeString(*deviceConfigStr)
	if err != nil {
		return "", fmt.Errorf("could not decode config from base64: %w", err)
	}

	deviceConfig := wgcfg.DeviceConfig{}
	if err := json.Unmarshal(configJSON, &deviceConfig); err != nil {
		return "", fmt.Errorf("could not unmarshal device config: %w", err)
	}

	return d.monitor.Up(deviceConfig, *uid)
}

func (d *Daemon) wgDown(args ...string) (err error) {
	flags := flag.NewFlagSet("", flag.ContinueOnError)
	interfaceName := flags.String("iface", "", "")
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}
	if *interfaceName == "" {
		return errors.New("-iface is required")
	}

	err = d.monitor.Down(*interfaceName)
	if err != nil {
		return fmt.Errorf("failed to down wg interface %s: %w", *interfaceName, err)
	}

	return nil
}

func (d *Daemon) wgStats(args ...string) (string, error) {
	flags := flag.NewFlagSet("", flag.ContinueOnError)
	interfaceName := flags.String("iface", "", "")
	if err := flags.Parse(args[1:]); err != nil {
		return "", err
	}
	if *interfaceName == "" {
		return "", errors.New("-iface is required")
	}
	stats, err := d.monitor.Stats(*interfaceName)
	if err != nil {
		return "", fmt.Errorf("could not get device stats for %s interface: %w", *interfaceName, err)
	}

	statsJSON, err := json.Marshal(stats)
	if err != nil {
		return "", fmt.Errorf("could not marshal stats to JSON: %w", err)
	}
	return string(statsJSON), nil
}
