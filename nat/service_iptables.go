package nat

import (
	"fmt"
	"os/exec"
)

type serviceIpTables struct {
	rules []RuleForwarding
}

func (service *serviceIpTables) Add(rule RuleForwarding) {
	service.rules = append(service.rules, rule)
}

func (service *serviceIpTables) Start() error {
	if err := service.enablePortForwarding(); err != nil {
		return err
	}
	if err := service.enableRules(); err != nil {
		service.disablePortForwarding()
		return err
	}

	return nil
}

func (service *serviceIpTables) Stop() error {
	if err := service.disableRules(); err != nil {
		return err
	}
	if err := service.disablePortForwarding(); err != nil {
		return err
	}

	return nil
}

func (service *serviceIpTables) enablePortForwarding() (err error) {
	cmd := exec.Command("echo", "1 > /proc/sys/net/ipv4/ip_forward")
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

func (service *serviceIpTables) disablePortForwarding() (err error) {
	cmd := exec.Command("echo", "0 > /proc/sys/net/ipv4/ip_forward")
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

func (service *serviceIpTables) enableRules() error {
	for _, rule := range service.rules {
		cmd := exec.Command("iptables", fmt.Sprintf(
			"--table nat --append POSTROUTING --source %s ! --destination %s --jump SNAT --to %s",
			rule.SourceAddress,
			rule.SourceAddress,
			rule.TargetIp,
		))
		if err := cmd.Run(); err != nil {
			return err
		}
	}
	return nil
}

func (service *serviceIpTables) disableRules() error {
	for _, rule := range service.rules {
		cmd := exec.Command("iptables", fmt.Sprintf(
			"--table nat --delete POSTROUTING --source %s ! --destination %s --jump SNAT --to %s",
			rule.SourceAddress,
			rule.SourceAddress,
			rule.TargetIp,
		))
		if err := cmd.Run(); err != nil {
			return err
		}
	}
	return nil
}
