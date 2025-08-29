/*
 * Copyright (C) 2025 The "MysteriumNetwork/node" Authors.
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

package dualproxyclient

import (
    "bufio"
    "context"
    "fmt"
    "net/http"
    "net/netip"
    "strings"
    "sync"
    "time"

    "github.com/rs/zerolog/log"
    "golang.zx2c4.com/wireguard/conn"
    "golang.zx2c4.com/wireguard/device"

    "github.com/mysteriumnetwork/node/services/wireguard/endpoint/netstack"
    "github.com/mysteriumnetwork/node/services/wireguard/endpoint/proxyclient"
    "github.com/mysteriumnetwork/node/services/wireguard/endpoint/socks5client"
    "github.com/mysteriumnetwork/node/services/wireguard/endpoint/userspace"
    "github.com/mysteriumnetwork/node/services/wireguard/wgcfg"
)

type client struct {
    mu         sync.Mutex
    Device     *device.Device
    httpClose  func() error
    socksClose func() error
}

// New client that serves HTTP and SOCKS5 proxies simultaneously.
func New() (*client, error) {
    log.Debug().Msg("Creating dual-proxy wg client (HTTP + SOCKS5)")
    return &client{}, nil
}

func (c *client) ReConfigureDevice(config wgcfg.DeviceConfig) error { return c.ConfigureDevice(config) }

func (c *client) ConfigureDevice(cfg wgcfg.DeviceConfig) error {
    localAddr, err := netip.ParseAddr(cfg.Subnet.IP.String())
    if err != nil {
        return fmt.Errorf("could not parse local addr: %w", err)
    }
    if len(cfg.DNS) == 0 {
        return fmt.Errorf("DNS addr list is empty")
    }
    dnsAddr, err := netip.ParseAddr(cfg.DNS[0])
    if err != nil {
        return fmt.Errorf("could not parse DNS addr: %w", err)
    }
    tunnel, tnet, err := netstack.CreateNetTUN([]netip.Addr{localAddr}, []netip.Addr{dnsAddr}, device.DefaultMTU)
    if err != nil {
        return fmt.Errorf("failed to create netstack device %s: %w", cfg.IfaceName, err)
    }

    logger := device.NewLogger(device.LogLevelVerbose, fmt.Sprintf("(%s) ", cfg.IfaceName))
    wgDevice := device.NewDevice(tunnel, conn.NewDefaultBind(), logger)

    log.Info().Msg("Applying interface configuration")
    if err := wgDevice.IpcSetOperation(bufio.NewReader(strings.NewReader(cfg.Encode()))); err != nil {
        wgDevice.Close()
        return fmt.Errorf("could not set device uapi config: %w", err)
    }

    log.Info().Msg("Bringing device up")
    wgDevice.Up()

    c.mu.Lock()
    c.Device = wgDevice
    c.mu.Unlock()

    log.Info().Msgf("Dual-proxy: starting proxies base_port=%d", cfg.ProxyPort)
    if err := c.startHTTPProxy(tnet, cfg.ProxyPort); err != nil {
        wgDevice.Close()
        return err
    }
    if err := c.startSOCKSProxy(tnet, cfg.ProxyPort); err != nil {
        c.stopHTTP()
        wgDevice.Close()
        return err
    }
    log.Info().Msg("Dual-proxy: proxies started")
    return nil
}

func (c *client) DestroyDevice(iface string) error { return c.Close() }

func (c *client) PeerStats(iface string) (wgcfg.Stats, error) {
    deviceState, err := userspace.ParseUserspaceDevice(c.Device.IpcGetOperation)
    if err != nil {
        return wgcfg.Stats{}, fmt.Errorf("could not parse device state: %w", err)
    }
    stats, statErr := userspace.ParseDevicePeerStats(deviceState)
    if err != nil {
        err = statErr
        log.Warn().Err(err).Msg("Failed to parse device stats, will try again")
    } else {
        return stats, nil
    }
    return wgcfg.Stats{}, fmt.Errorf("could not parse device state: %w", err)
}

func (c *client) Close() error {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.stopSOCKS()
    c.stopHTTP()
    if c.Device != nil {
        go func() {
            time.Sleep(2 * time.Minute)
            c.Device.Close()
        }()
    }
    return nil
}

func (c *client) startHTTPProxy(tnet *netstack.Net, httpPort int) error {
    if httpPort <= 0 {
        return fmt.Errorf("http proxy port is not set")
    }
    server := http.Server{
        Addr:              fmt.Sprintf(":%d", httpPort),
        Handler:           proxyclient.NewProxyHandler(60*time.Second, tnet),
        ReadTimeout:       0,
        ReadHeaderTimeout: 0,
        WriteTimeout:      0,
        IdleTimeout:       0,
    }

    log.Info().Msgf("Starting HTTP proxy server at :%d ...", httpPort)
    c.httpClose = func() error {
        ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
        defer cancel()
        server.Shutdown(ctx)
        return server.Close()
    }
    go func() {
        err := server.ListenAndServe()
        if err != nil {
            log.Error().Err(err).Msg("Shutting down HTTP proxy server...")
        }
    }()
    return nil
}

func (c *client) startSOCKSProxy(tnet *netstack.Net, basePort int) error {
    socksPort := basePort + 1
    if socksPort <= 1 { // base not set or 0, fall back to default
        socksPort = 1080
    }
    srv := &socks5client.Server{Dialer: socks5client.NewNetstackAdapter(tnet)}
    addr := fmt.Sprintf(":%d", socksPort)
    log.Info().Msgf("Starting SOCKS5 proxy server at %s ...", addr)
    done := make(chan struct{})
    go func() {
        _ = srv.Serve(addr)
        close(done)
    }()
    c.socksClose = func() error { srv.Close(); <-done; return nil }
    return nil
}

func (c *client) stopHTTP() {
    if c.httpClose != nil {
        _ = c.httpClose()
        c.httpClose = nil
    }
}

func (c *client) stopSOCKS() {
    if c.socksClose != nil {
        _ = c.socksClose()
        c.socksClose = nil
    }
}
