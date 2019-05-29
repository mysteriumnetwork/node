package firewall

import log "github.com/cihub/seelog"

type NoopBlocker struct {
	LogPrefix string
}

func (nb NoopBlocker) BlockNonTunnelTraffic(scope Scope) (RemoveRule, error) {
	log.Info(nb.LogPrefix, "Non tunneled traffic block requested. Scope: ", scope)
	return nb.logRemoval("Block for scope: ", scope, " removed"), nil
}

func (nb NoopBlocker) AllowURLAccess(url string) (RemoveRule, error) {
	log.Info(nb.LogPrefix, "Allow ", url, " access")
	return nb.logRemoval("Rule for ", url, " removed"), nil
}

func (nb NoopBlocker) AllowIPAccess(ip string) (RemoveRule, error) {
	log.Info(nb.LogPrefix, "Allow ", ip, " access")
	return nb.logRemoval("Rule for ip: ", ip, " removed"), nil
}

func (nb NoopBlocker) logRemoval(vals ...interface{}) RemoveRule {
	return func() {
		vals := append([]interface{}{nb.LogPrefix}, vals...)
		log.Info(vals...)
	}
}

var _ Blocker = NoopBlocker{}
