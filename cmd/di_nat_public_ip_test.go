/*
 * Copyright (C) 2026 The "MysteriumNetwork/node" Authors.
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

package cmd

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/mysteriumnetwork/node/core/ip"
	"github.com/mysteriumnetwork/node/eventbus"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/nat/event"
)

const testIdentity = "0x1111111111111111111111111111111111111111"

func natEventsFor(t *testing.T, resolver ip.Resolver) <-chan event.Event {
	di := &Dependencies{EventBus: eventbus.New(), IPResolver: resolver}

	events := make(chan event.Event, 1)
	assert.NoError(t, di.EventBus.Subscribe(event.AppTopicTraversal, func(e event.Event) { events <- e }))
	assert.NoError(t, di.subscribeNATStatusForPublicIP())

	di.EventBus.Publish(identity.AppTopicIdentityUnlock, identity.AppEventIdentityUnlock{ID: identity.FromAddress(testIdentity)})
	return events
}

// The "public_ip" NAT event feeds a quality metric that is signed by its owner,
// so it must carry the unlocked identity instead of an empty ID.
func TestNATStatusForPublicIP_PublishesWithUnlockedIdentity(t *testing.T) {
	events := natEventsFor(t, ip.NewResolverMockMultiple("1.2.3.4", "1.2.3.4"))

	select {
	case e := <-events:
		assert.Equal(t, testIdentity, e.ID)
		assert.Equal(t, "public_ip", e.Stage)
		assert.True(t, e.Successful)
	case <-time.After(2 * time.Second):
		t.Fatal("expected public_ip NAT event after identity unlock")
	}
}

func TestNATStatusForPublicIP_SkipsWhenBehindNAT(t *testing.T) {
	events := natEventsFor(t, ip.NewResolverMockMultiple("192.168.1.10", "1.2.3.4"))

	select {
	case e := <-events:
		t.Fatalf("unexpected NAT event behind NAT: %+v", e)
	case <-time.After(300 * time.Millisecond):
	}
}
