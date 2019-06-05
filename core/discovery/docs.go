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

/*
Package discovery is responsible for:
 - handling discovery related communications
 - keeping track of currently active proposals in local storage
 - finding and filtering proposals in local storage


Proposal storage filtering can be done with ProposalReducer callbacks.
ProposalReducer is simple callback which is run against list of proposals.
Proposal storage filtering is performed with discovery.Finder:
```
finder := discovery.NewFinder(discovery.NewStorage())
proposal, err := finder.GetProposal(market.ProposalID{ServiceType: "streaming", ProviderID: "0x1"})
proposals, err = finder.MatchProposals(reducer.Equal(reducer.ProviderID, "0x2"))
```

- Most simple noop reducer:
```
finder.MatchProposals(
  reducer.All(),
)
```
- By proposal field:
```
finder.MatchProposals(
  reducer.EqualString(reducer.ProviderID, "0x1"),
)
```
- By custom callback:
```
finder.MatchProposals(func(proposal market.ServiceProposal) bool {
  return proposal.ProviderID == "0x3" || proposal.ServiceType == "wireguard"
})
```
- By several conditions:
```
finder.MatchProposals(reducer.And(
  reducer.EqualString(reducer.ProviderID, "0x1"),
  reducer.InString(reducer.ServiceType, "openvpn", "wireguard")),
  reducer.AccessPolicy("mysterium", ""),
))
```
- By nesting several conditions:
```
finder.MatchProposals(reducer.Or(
  reducer.And(
    reducer.EqualString(reducer.ServiceType, "wireguard"),
    reducer.InString(reducer.LocationCountry, "US", "CA"),
    reducer.EqualString(reducer.LocationType, "residential"),
  ),
  reducer.And(
    reducer.EqualString(reducer.ServiceType, "openvpn"),
    reducer.Or(
      reducer.EqualString(reducer.LocationCountry, "US")),
      reducer.EqualString(reducer.LocationCountry, "CA")),
    ),
    reducer.Not(reducer.EqualString(reducer.LocationType, "datacenter")),
  )
))
```
*/

package discovery
