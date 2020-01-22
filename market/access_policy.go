/*
 * Copyright (C) 2020 The "MysteriumNetwork/node" Authors.
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

package market

const (
	// AccessPolicyTypeIdentity Explicitly allow just specific identities ("0xd1faed693fec75389c3d1e59b863e4835ac6f5d1")
	AccessPolicyTypeIdentity = "identity"
	// AccessPolicyTypeDNSHostname Explicitly allow just specific hostname ("ipinfo.io")
	AccessPolicyTypeDNSHostname = "dns_hostname"
	// AccessPolicyTypeDNSZone Explicitly allow just specific DNS zone ("example.com" matches "example.com" and all of its subdomains)
	AccessPolicyTypeDNSZone = "dns_zone"
)

// AccessPolicy represents the access controls for proposal
type AccessPolicy struct {
	ID     string `json:"id"`
	Source string `json:"source"`
}

// AccessPolicyRuleSet represents named list with rules specifying whether access is allowed
type AccessPolicyRuleSet struct {
	ID          string       `json:"id"`
	Title       string       `json:"title"`
	Description string       `json:"description"`
	Allow       []AccessRule `json:"allow"`
}

// AccessRule represents rule specifying whether connection should be allowed
type AccessRule struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}
