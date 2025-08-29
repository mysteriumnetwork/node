package smartclient

import (
    "fmt"

    "github.com/rs/zerolog/log"

    "github.com/mysteriumnetwork/node/config"
    "github.com/mysteriumnetwork/node/services/wireguard/endpoint/dvpnclient"
    "github.com/mysteriumnetwork/node/services/wireguard/endpoint/kernelspace"
    netstack_provider "github.com/mysteriumnetwork/node/services/wireguard/endpoint/netstack-provider"
    "github.com/mysteriumnetwork/node/services/wireguard/endpoint/remoteclient"
    "github.com/mysteriumnetwork/node/services/wireguard/endpoint/userspace"
    "github.com/mysteriumnetwork/node/services/wireguard/endpoint/dualproxyclient"
    "github.com/mysteriumnetwork/node/services/wireguard/wgcfg"
)

// WgClient minimal interface used by endpoint.
type WgClient interface {
    ConfigureDevice(config wgcfg.DeviceConfig) error
    ReConfigureDevice(config wgcfg.DeviceConfig) error
    DestroyDevice(name string) error
    PeerStats(iface string) (wgcfg.Stats, error)
    Close() error
}

// Client defers the concrete client selection until ConfigureDevice,
// so we can choose based on cfg.ProxyPort reliably.
type Client struct {
    impl WgClient
}

func New() *Client { return &Client{} }

func (c *Client) choose(cfg wgcfg.DeviceConfig) (WgClient, error) {
    // dVPN mode takes precedence if enabled globally.
    if config.GetBool(config.FlagDVPNMode) {
        log.Info().Msg("smartclient: using dvpnclient")
        return dvpnclient.New()
    }
    // If a proxy port is provided by the consumer, run dual-proxy (HTTP+SOCKS5).
    if cfg.ProxyPort > 0 {
        log.Info().Msgf("smartclient: proxy requested, starting dual-proxy (base=%d)", cfg.ProxyPort)
        return dualproxyclient.New()
    }
    // Otherwise follow original selection logic.
    if config.GetBool(config.FlagUserspace) {
        log.Info().Msg("smartclient: userspace enabled, using netstack-provider")
        return netstack_provider.New()
    }
    if config.GetBool(config.FlagUserMode) {
        log.Info().Msg("smartclient: usermode enabled, using remoteclient")
        return remoteclient.New()
    }
    // Try kernelspace, fallback to userspace
    kc, err := kernelspace.NewWireguardClient()
    if err == nil {
        log.Info().Msg("smartclient: using kernelspace client")
        return kc, nil
    }
    log.Info().Msg("smartclient: kernelspace unsupported, using userspace client")
    return userspace.NewWireguardClient()
}

func (c *Client) ConfigureDevice(cfg wgcfg.DeviceConfig) error {
    if c.impl == nil {
        impl, err := c.choose(cfg)
        if err != nil {
            return fmt.Errorf("smartclient: choose failed: %w", err)
        }
        c.impl = impl
    }
    return c.impl.ConfigureDevice(cfg)
}

func (c *Client) ReConfigureDevice(cfg wgcfg.DeviceConfig) error {
    if c.impl == nil {
        if err := c.ConfigureDevice(cfg); err != nil { return err }
        return nil
    }
    return c.impl.ReConfigureDevice(cfg)
}

func (c *Client) DestroyDevice(name string) error {
    if c.impl == nil { return nil }
    return c.impl.DestroyDevice(name)
}

func (c *Client) PeerStats(iface string) (wgcfg.Stats, error) {
    if c.impl == nil { return wgcfg.Stats{}, fmt.Errorf("smartclient: not configured") }
    return c.impl.PeerStats(iface)
}

func (c *Client) Close() error {
    if c.impl == nil { return nil }
    return c.impl.Close()
}

