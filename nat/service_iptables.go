package nat

import (
	"fmt"
	"os/exec"

	log "github.com/cihub/seelog"
)

const NAT_LOG_PREFIX = "[nat] "

type serviceIpTables struct {
	rules []RuleForwarding
}

func (service *serviceIpTables) Add(rule RuleForwarding) {
	service.rules = append(service.rules, rule)
}

func (service *serviceIpTables) Start() error {
	if err := service.enableIPForwarding(); err != nil {
		return err
	}
	if err := service.enableRules(); err != nil {
		service.disableIPForwarding()
		return err
	}

	return nil
}

func (service *serviceIpTables) Stop() error {
	if err := service.disableRules(); err != nil {
		return err
	}
	if err := service.disableIPForwarding(); err != nil {
		return err
	}

	return nil
}

func (service *serviceIpTables) enableIPForwarding() (err error) {
	cmd := exec.Command("sysctl", "-w", "net.ipv4.ip_forward=1")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Failed to enable IP forwarding: %s", err)
	}

	log.Info(NAT_LOG_PREFIX, "IP forwarding enabled")
	return nil
}

func (service *serviceIpTables) disableIPForwarding() (err error) {
	cmd := exec.Command("sysctl", "-w", "net.ipv4.ip_forward=0")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Failed to disable IP forwarding. %s", err)
	}

	log.Info(NAT_LOG_PREFIX, "IP forwarding disabled")
	return nil
}

func (service *serviceIpTables) enableRules() error {
	for _, rule := range service.rules {
		cmd := exec.Command(
			"iptables",
			"--table", "nat",
			"--append", "POSTROUTING",
			"--source", rule.SourceAddress,
			"!", "--destination", rule.SourceAddress,
			"--jump", "SNAT",
			"--to", rule.TargetIp,
		)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("Failed to create ip forwarding rule: %s. %s", cmd.Args, err.Error())
		}
		log.Info(NAT_LOG_PREFIX, "Forwarding packets from '", rule.SourceAddress, "' to IP: ", rule.TargetIp)
	}
	return nil
}

func (service *serviceIpTables) disableRules() error {
	for _, rule := range service.rules {
		cmd := exec.Command(
			"iptables",
			"--table", "nat",
			"--delete", "POSTROUTING",
			"--source", rule.SourceAddress,
			"!", "--destination", rule.SourceAddress,
			"--jump", "SNAT",
			"--to", rule.TargetIp,
		)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("Failed to delete ip forwarding rule: %s. %s", cmd.Args, err.Error())
		}
		log.Info(NAT_LOG_PREFIX, "Stopped forwarding packets from '", rule.SourceAddress, "' to IP: ", rule.TargetIp)
	}
	return nil
}
