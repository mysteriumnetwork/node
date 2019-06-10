package iptables

import "strconv"

type chainInfo struct {
	chainName string
	action    []string
	ruleArgs  []string
}

func appendTo(chainName string) chainInfo {
	return chainInfo{
		chainName: chainName,
		action:    []string{appendRule, chainName},
	}
}

func insertAt(chainName string, line int) chainInfo {
	return chainInfo{
		chainName: chainName,
		action:    []string{insertRule, chainName, strconv.Itoa(line)},
	}
}

func (chain chainInfo) ruleSpec(args ...string) chainInfo {
	chain.ruleArgs = args
	return chain
}

func (chain chainInfo) applyArgs() []string {
	return append(chain.action, chain.ruleArgs...)
}

func (chain chainInfo) removeArgs() []string {
	return append([]string{removeRule, chain.chainName}, chain.ruleArgs...)
}
