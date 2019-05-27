package firewall

type RemoveRule func()

type Scope string

const (
	Global  Scope = "global"
	Session Scope = "session"
	none    Scope = "none"
)

var currentBlocker Blocker = NoopBlocker{
	LogPrefix: "[Noop firewall] ",
}

func Configure(blocker Blocker) {
	currentBlocker = blocker
}

func BlockNonTunnelTraffic(scope Scope) (RemoveRule, error) {
	return currentBlocker.BlockNonTunnelTraffic(scope)
}

func AllowURLAccess(url string) (RemoveRule, error) {
	return currentBlocker.AllowURLAccess(url)
}

func AllowIPAccess(ip string) (RemoveRule, error) {
	return currentBlocker.AllowIPAccess(ip)
}

type Blocker interface {
	BlockNonTunnelTraffic(scope Scope) (RemoveRule, error)
	AllowURLAccess(url string) (RemoveRule, error)
	AllowIPAccess(ip string) (RemoveRule, error)
}
