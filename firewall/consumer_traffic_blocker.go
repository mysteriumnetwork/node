package firewall

var trackingBlocker = newTrackingBlocker()

// Configure blocker with specified actual Vendor implementation
func Configure(vendor Vendor) {
	trackingBlocker.SwitchVendor(vendor)
}

// BlockNonTunnelTraffic effectively disallows any outgoing traffic from consumer node with specified scope
func BlockNonTunnelTraffic(scope Scope) (RemoveRule, error) {
	return trackingBlocker.BlockOutgoingTraffic(scope)
}

// AllowURLAccess adds exception to blocked traffic for specified URL (host part is usually taken)
func AllowURLAccess(URLs ...string) (RemoveRule, error) {
	return trackingBlocker.AllowURLAccess(URLs...)
}

// AllowIPAccess adds IP based exception to underlying blocker implementation
func AllowIPAccess(ip string) (RemoveRule, error) {
	return trackingBlocker.AllowIPAccess(ip)
}

func Reset() {
	trackingBlocker.vendor.Reset()
}

// Vendor interface neededs to be satisfied by any implementations which provide firewall capabilities, like iptables
type Vendor interface {
	BlockOutgoingTraffic() (RemoveRule, error)
	AllowIPAccess(ip string) (RemoveRule, error)
	Reset()
}
