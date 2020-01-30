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

// Rule is a packet filter rule for IPTables.
type Rule struct {
	chainName string
	action    []string
	ruleSpec  []string
}

// AppendTo creates a new rule to be appended to the specified chain.
func AppendTo(chainName string) Rule {
	return Rule{
		chainName: chainName,
		action:    []string{"-A", chainName},
	}
}

// InsertAt creates a new rule to be inserted into the specified chain.
func InsertAt(chainName string, line int) Rule {
	return Rule{
		chainName: chainName,
		action:    []string{"-I", chainName, strconv.Itoa(line)},
	}
}

// RuleSpec sets the rule specification (see `man iptables`).
func (r Rule) RuleSpec(spec ...string) Rule {
	r.ruleSpec = spec
	return r
}

// ApplyArgs returns an argument list to be passed to the iptables executable to APPLY the rule.
func (r Rule) ApplyArgs() []string {
	return append(r.action, r.ruleSpec...)
}

// RemoveArgs returns an argument list to be passed to the iptables executable to REMOVE the rule.
func (r Rule) RemoveArgs() []string {
	return append([]string{"-D", r.chainName}, r.ruleSpec...)
}

// Equals checks if two Rules are equal.
func (r Rule) Equals(another Rule) bool {
	return r.chainName == another.chainName &&
		equalStringSlice(r.ruleSpec, another.ruleSpec)
}

func equalStringSlice(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
