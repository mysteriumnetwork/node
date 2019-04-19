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

package config

import (
	"encoding/json"

	"github.com/mysteriumnetwork/node/services"
	"github.com/mysteriumnetwork/node/services/openvpn"
	"github.com/pkg/errors"
)

// ConsumerConfigParser parses consumer configs
type ConsumerConfigParser struct {
}

// NewConfigParser returns a new ConsumerConfigParser
func NewConfigParser() *ConsumerConfigParser {
	return &ConsumerConfigParser{}
}

// Parse parses the given configuration
func (c *ConsumerConfigParser) Parse(config json.RawMessage) (ip string, port int, serviceType services.ServiceType, err error) {
	// TODO: since we are getting json.RawMessage here and not interface{} type not sure how to handle multiple services
	// since NATPinger is one for all services and we get config from communication channel where service type is not know yet.
	var cfg openvpn.ConsumerConfig
	err = json.Unmarshal(config, &cfg)
	if err != nil {
		return "", 0, "", errors.Wrap(err, "parsing consumer address:port failed")
	}

	if cfg.IP == nil {
		return "", 0, "", errors.New("remote party does not support NAT hole punching, IP:PORT is missing")
	}
	return *cfg.IP, cfg.Port, openvpn.ServiceType, nil
}
