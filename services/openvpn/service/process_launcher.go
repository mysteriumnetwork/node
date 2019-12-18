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

package service

import (
	"github.com/mysteriumnetwork/go-openvpn/openvpn"
	"github.com/mysteriumnetwork/go-openvpn/openvpn/middlewares/server/auth"
	"github.com/mysteriumnetwork/go-openvpn/openvpn/middlewares/server/bytecount"
	"github.com/mysteriumnetwork/go-openvpn/openvpn/middlewares/server/filter"
	"github.com/mysteriumnetwork/go-openvpn/openvpn/middlewares/state"
	"github.com/mysteriumnetwork/node/core/node"
	openvpn_service "github.com/mysteriumnetwork/node/services/openvpn"
	openvpn_session "github.com/mysteriumnetwork/node/services/openvpn/session"
)

type processLauncher struct {
	opts             node.Options
	sessionValidator *openvpn_session.Validator
	statsCallback    func(count bytecount.SessionByteCount)
}

func newProcessLauncher(opts node.Options, sessionValidator *openvpn_session.Validator, statsCallback func(bytecount.SessionByteCount)) *processLauncher {
	return &processLauncher{
		opts:             opts,
		sessionValidator: sessionValidator,
		statsCallback:    statsCallback,
	}
}

type launchOpts struct {
	config                   *openvpn_service.ServerConfig
	filterAllow, filterBlock []string
	stateChannel             chan openvpn.State
}

func (p *processLauncher) launch(opts launchOpts) openvpn.Process {
	stateCallback := func(state openvpn.State) {
		opts.stateChannel <- state
		//this is the last state - close channel (according to best practices of go - channel writer controls channel)
		if state == openvpn.ProcessExited {
			close(opts.stateChannel)
		}
	}

	return openvpn.CreateNewProcess(
		p.opts.Openvpn.BinaryPath(),
		opts.config.GenericConfig,
		filter.NewMiddleware(opts.filterAllow, opts.filterBlock),
		auth.NewMiddleware(p.sessionValidator.Validate),
		state.NewMiddleware(stateCallback),
		bytecount.NewMiddleware(p.statsCallback, statisticsReportingIntervalInSeconds),
	)
}
