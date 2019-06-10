package iptables

import (
	"bufio"
	"bytes"
	"os/exec"
	"strings"

	log "github.com/cihub/seelog"
	"github.com/mysteriumnetwork/node/firewall"
	"github.com/pkg/errors"
)

const (
	logPrefix = "[iptables] "

	outputChain     = "OUTPUT"
	killswitchChain = "CONSUMER_KILL_SWITCH"

	addChain         = "-N"
	appendRule       = "-A"
	insertRule       = "-I"
	listRules        = "-S"
	removeRule       = "-D"
	removeChainRules = "-F"
	removeChain      = "-X"

	jumpTo        = "-j"
	sourceIP      = "-s"
	destinationIP = "-d"
	module        = "-m"

	conntrack    = "conntrack"
	ct_state     = "--ctstate"
	ct_state_new = "NEW"

	reject = "REJECT"
	accept = "ACCEPT"

	version = "--version"
)

var iptablesExec = func(args ...string) ([]string, error) {
	args = append([]string{"/sbin/iptables"}, args...)
	log.Trace(logPrefix, "[cmd] ", args)
	output, err := exec.Command("sudo", args...).Output()
	if err != nil {
		log.Trace(logPrefix, "[cmd error] ", err)
		return nil, errors.Wrap(err, "iptables cmd error")
	}
	outputScanner := bufio.NewScanner(bytes.NewBuffer(output))
	var lines []string
	for outputScanner.Scan() {
		lines = append(lines, outputScanner.Text())
	}
	return lines, outputScanner.Err()
}

func checkVersion() error {
	output, err := iptablesExec(version)
	if err != nil {
		return err
	}
	for _, line := range output {
		log.Info(logPrefix, "[version check] ", line)
	}
	return nil
}

func cleanupStaleRules() error {
	rules, err := iptablesExec(listRules, outputChain)
	if err != nil {
		return err
	}
	for _, rule := range rules {
		//detect if any references exist in OUTPUT chain like -j CONSUMER_KILL_SWITCH
		if strings.HasSuffix(rule, killswitchChain) {
			deleteRule := strings.Replace(rule, appendRule, removeRule, 1)
			if _, err := iptablesExec(deleteRule); err != nil {
				return err
			}
		}
	}

	if _, err := iptablesExec(listRules, killswitchChain); err != nil {
		//error means no such chain - log error just in case and bail out
		log.Info(logPrefix, "[setup] Got error while listing kill switch chain rules. Probably nothing to worry about. Err: ", err)
		return nil
	}

	if _, err := iptablesExec(removeChainRules, killswitchChain); err != nil {
		return err
	}

	_, err = iptablesExec(removeChain, killswitchChain)
	return err
}

func setupKillSwitchChain() error {
	if _, err := iptablesExec(addChain, killswitchChain); err != nil {
		return err
	}
	// by default all packets going to kill switch chain are rejected
	if _, err := iptablesExec(appendRule, killswitchChain, module, conntrack, ct_state, ct_state_new, jumpTo, reject); err != nil {
		return err
	}
	return nil
}

type Iptables struct {
	outboundIP string
}

func (b Iptables) BlockOutgoingTraffic() (firewall.RemoveRule, error) {
	return addRuleWithRemoval(appendTo(outputChain).ruleSpec(sourceIP, b.outboundIP, jumpTo, killswitchChain))
}

func New(outboundIP string) *Iptables {
	return &Iptables{
		outboundIP: outboundIP,
	}
}

func (b Iptables) Setup() error {
	if err := checkVersion(); err != nil {
		return err
	}

	if err := cleanupStaleRules(); err != nil {
		return err
	}

	return setupKillSwitchChain()
}

func (Iptables) Reset() {
	if err := cleanupStaleRules(); err != nil {
		_ = log.Warn(logPrefix, "Error cleaning up iptables rules, you might want to do it yourself: ", err)
	}
}

func addRuleWithRemoval(chain chainInfo) (firewall.RemoveRule, error) {
	if _, err := iptablesExec(chain.applyArgs()...); err != nil {
		return nil, err
	}
	return func() {
		_, err := iptablesExec(chain.removeArgs()...)
		if err != nil {
			_ = log.Warn(logPrefix, "Error executing rule: ", chain.removeArgs(), " you might wanna do it yourself. Error was: ", err)
		}
	}, nil
}

func (Iptables) AllowIPAccess(ip string) (firewall.RemoveRule, error) {
	return addRuleWithRemoval(insertAt(killswitchChain, 1).ruleSpec(destinationIP, ip, jumpTo, accept))
}

var _ firewall.Vendor = (*Iptables)(nil)
