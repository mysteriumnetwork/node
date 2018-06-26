/*
 * Copyright (C) 2017 The "MysteriumNetwork/node" Authors.
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

package openvpn

type State string

const ConnectingState = State("CONNECTING")
const WaitState = State("WAIT")
const AuthenticatingState = State("AUTH")
const GetConfigState = State("GET_CONFIG")
const AssignIpState = State("ASSIGN_IP")
const AddRoutesState = State("ADD_ROUTES")
const ConnectedState = State("CONNECTED")
const ReconnectingState = State("RECONNECTING")
const ExitingState = State("EXITING")

// two "fake" states which has no description in openvpn management interface documentation
// ProcessStarted state is reported by state middleware when middleware start method itself is called
// it means that process successfully connected to management interface
const ProcessStarted = State("PROCESS_STARTED")

// ProcessExited state is reported on state middleware stop method to indicate that process disconnected from
// interface and this is LAST state to report
const ProcessExited = State("PROCESS_EXITED")

// UnknownState is reported when state middleware cannot parse state from string (i.e. it's undefined in list above),
// usually that means that newer openvpn version reports something extra
const UnknownState = State("UNKNOWN")
