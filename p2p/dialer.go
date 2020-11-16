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

package p2p

import (
	"context"

	"github.com/mysteriumnetwork/node/core/ip"
	"github.com/mysteriumnetwork/node/core/port"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/trace"
)

// Dialer knows how to exchange p2p keys and encrypted configuration and creates ready to use p2p channels.
type Dialer interface {
	// Dial exchanges p2p configuration via broker, performs NAT pinging if needed
	// and create p2p channel which is ready for communication.
	Dial(ctx context.Context, consumerID, providerID identity.Identity, serviceType string, contactDef []ContactDefinition, tracer *trace.Tracer) (Channel, error)
}

// NewDialer creates new dialer for both HTTP and NATS brokers. HTTP broker has a higher priority and NATS is used only when HTTP broker unavailable.
func NewDialer(broker brokerConnector, signer identity.SignerFactory, verifier identity.Verifier, ipResolver ip.Resolver, consumerPinger natConsumerPinger, portPool port.ServicePortSupplier) Dialer {
	return &dialer{
		nats: &dialerNATS{
			broker:         broker,
			ipResolver:     ipResolver,
			signer:         signer,
			verifier:       verifier,
			portPool:       portPool,
			consumerPinger: consumerPinger,
		},
		http: &dialerHTTP{
			ipResolver:     ipResolver,
			signer:         signer,
			verifier:       verifier,
			portPool:       portPool,
			consumerPinger: consumerPinger,
		},
	}
}

type dialer struct {
	nats *dialerNATS
	http *dialerHTTP
}

func (d *dialer) Dial(ctx context.Context, consumerID, providerID identity.Identity, serviceType string, contactDef []ContactDefinition, tracer *trace.Tracer) (Channel, error) {
	for _, def := range contactDef {
		if def.Type == ContactTypeHTTPv1 {
			return d.http.Dial(ctx, consumerID, providerID, serviceType, def, tracer)
		}
	}

	for _, def := range contactDef {
		if def.Type == ContactTypeNATSv1 {
			return d.nats.Dial(ctx, consumerID, providerID, serviceType, def, tracer)
		}
	}

	return nil, ErrContactNotFound
}
