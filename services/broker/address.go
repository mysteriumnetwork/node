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

package broker

import (
	"fmt"
	"sync"
	"time"

	"github.com/mysteriumnetwork/node/config"
	"github.com/mysteriumnetwork/node/core/ip"
	"github.com/mysteriumnetwork/node/eventbus"
)

var (
	connMu sync.Mutex
)

type BrokerAddress struct {
	EventBus   eventbus.EventBus
	IPResolver ip.Resolver
	address    string
}

func (b *BrokerAddress) SetAddress(address string) {
	b.address = address
}

func (b *BrokerAddress) GetAddress() string {
	for start := time.Now(); start.Add(10 * time.Second).Before(time.Now()); {
		if len(b.address) > 0 {
			break
		}

		time.Sleep(time.Second)
	}

	return b.address
}

func (b *BrokerAddress) Address(serviceType, providerID string, f func()) string {
	connMu.Lock()
	defer connMu.Unlock()

	if serviceType == ServiceType {
		ip, err := b.IPResolver.GetPublicIP()
		if err != nil {
			return err.Error()
		}

		b.SetAddress(fmt.Sprintf("http://%s:%d/00000000-0000-0000-0000-000000000000", ip, config.GetInt(config.FlagBrokerPort)))
	}

	if address := b.GetAddress(); len(address) > 0 {
		return address
	}

	f()

	if address := b.GetAddress(); len(address) > 0 {
		return address
	}

	return ""
}
