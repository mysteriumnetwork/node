/*
 * Copyright (C) 2019 The "MysteriumNetwork/node" Authors.
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
