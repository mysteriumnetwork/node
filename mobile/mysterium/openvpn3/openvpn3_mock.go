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

package openvpn3

import "errors"

// This package is a mock of github.com/mysteriumnetwork/go-openvpn. It is mostly useful only when
// you want to quickly build and test your changes related to mobile and you don't need openvpn or just working
// with wireguard related logic.
//
// Build Mysterium.aar package locally: GO111MODULE=off gomobile bind -target=android -o ./build/package/Mysterium.aar github.com/mysteriumnetwork/node/mobile/mysterium

// NewMobileSession returns mock session
func NewMobileSession(config Config, userCredentials UserCredentials, callbacks MobileSessionCallbacks, tunSetup TunnelSetup) *Session {
	return &Session{}
}

// MobileSessionCallbacks represents mock callbacks
type MobileSessionCallbacks interface {
	EventConsumer
	Logger
	StatsConsumer
}

// EventConsumer mock
type EventConsumer interface {
	OnEvent(Event)
}

// Logger represents the logger
type Logger interface {
	Log(string)
}

// StatsConsumer consumes the bytes/in out statistics
type StatsConsumer interface {
	OnStats(Statistics)
}

// Session mock
type Session struct {
}

// Start mock
func (s Session) Start() {
}

// Wait mock
func (s Session) Wait() error {
	return errors.New("using mock openvpn3")
}

// Reconnect mock
func (s Session) Reconnect(afterSeconds int) error {
	return nil
}

// Stop mock
func (s Session) Stop() {
}

// Event mock
type Event struct {
	Name string
}

// Statistics mock
type Statistics struct {
	BytesIn  uint64
	BytesOut uint64
}

// TunnelSetup mock
type TunnelSetup interface {
	NewBuilder() bool
	SetLayer(layer int) bool
	SetRemoteAddress(ipAddress string, ipv6 bool) bool
	AddAddress(address string, prefixLength int, gateway string, ipv6 bool, net30 bool) bool
	SetRouteMetricDefault(metric int) bool
	RerouteGw(ipv4 bool, ipv6 bool, flags int) bool
	AddRoute(address string, prefixLength int, metric int, ipv6 bool) bool
	ExcludeRoute(address string, prefixLength int, metric int, ipv6 bool) bool
	AddDnsServer(address string, ipv6 bool) bool
	AddSearchDomain(domain string) bool
	SetMtu(mtu int) bool
	SetSessionName(name string) bool
	AddProxyBypass(bypassHost string) bool
	SetProxyAutoConfigUrl(url string) bool
	SetProxyHttp(host string, port int) bool
	SetProxyHttps(host string, port int) bool
	AddWinsServer(address string) bool
	SetBlockIpv6(ipv6Block bool) bool
	SetAdapterDomainSuffix(name string) bool
	Establish() (int, error)
	Persist() bool
	EstablishLite()
	Teardown(disconnect bool)
	SocketProtect(socket int) bool
}

// Config mock
type Config struct {
	GuiVersion      string
	CompressionMode string
	ConnTimeout     int
}

// NewConfig mock
func NewConfig(_ string) Config {
	return Config{}
}

// UserCredentials mock
type UserCredentials struct {
	Username string
	Password string
}
