package iptables

import (
	"bufio"
	"bytes"
	"os/exec"
	"strings"

	"github.com/pkg/errors"

	"github.com/mysteriumnetwork/node/firewall"

	log "github.com/cihub/seelog"
)

const (
	logPrefix = "[iptables] "

	outputChain     = "OUTPUT"
	killswitchChain = "CONSUMER_KILL_SWITCH"

	addChain         = "-N"
	addRule          = "-A"
	listRules        = "-S"
	removeRule       = "-D"
	removeChainRules = "-F"
	removeChain      = "-X"

	jumpTo        = "-j"
	sourceIP      = "-s"
	destinationIP = "-d"

	reject = "REJECT"
	accept = "ACCEPT"

	version = "--version"
)

var iptablesExec = func(args ...string) ([]string, error) {
	args = append([]string{"/sbin/iptables"}, args...)
	output, err := exec.Command("sudo", args...).Output()
	if err != nil {
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
			deleteRule := strings.Replace(rule, addRule, removeRule, 1)
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
	if _, err := iptablesExec(addRule, killswitchChain, jumpTo, reject); err != nil {
		return err
	}
	return nil
}

type IptablesBlocker struct {
	outboundIP string
}

func (ib IptablesBlocker) BlockOutgoingTraffic() (firewall.RemoveRule, error) {
	return iptablesAddWithRemoval(outputChain, sourceIP, ib.outboundIP, jumpTo, killswitchChain)
}

func NewBlocker(outboundIP string) *IptablesBlocker {
	return &IptablesBlocker{
		outboundIP: outboundIP,
	}
}

func (ib IptablesBlocker) Setup() error {
	if err := checkVersion(); err != nil {
		return err
	}

	if err := cleanupStaleRules(); err != nil {
		return err
	}

	return setupKillSwitchChain()
}

func (IptablesBlocker) Reset() {
	if err := cleanupStaleRules(); err != nil {
		_ = log.Warn(logPrefix, "Error cleaning up iptables rules, you might want to do it yourself: ", err)
	}
}

func iptablesAddWithRemoval(args ...string) (firewall.RemoveRule, error) {
	addRule := append([]string{addRule}, args...)
	removeRule := append([]string{removeRule}, args...)
	if _, err := iptablesExec(addRule...); err != nil {
		return nil, err
	}
	return func() {
		_, err := iptablesExec(removeRule...)
		if err != nil {
			_ = log.Warn(logPrefix, "Error deleting rule: ", removeRule, " you might wanna do it yourself. Error was: ", err)
		}
	}, nil
}

func (IptablesBlocker) AllowIPAccess(ip string) (firewall.RemoveRule, error) {
	return iptablesAddWithRemoval(killswitchChain, destinationIP, ip, jumpTo, accept)
}

var _ firewall.BlockVendor = (*IptablesBlocker)(nil)
