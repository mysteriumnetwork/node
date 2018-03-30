package server

import (
	"github.com/stretchr/testify/suite"
	"github.com/mysterium/node/server"
	"github.com/mysterium/node/identity"
	dto_discovery "github.com/mysterium/node/service_discovery/dto"
	"github.com/stretchr/testify/assert"
)

type testContext struct {
	suite.Suite
}

var (
	myID                  = identity.FromAddress("identity-1")
	activeProviderID      = identity.FromAddress("vpn-node-1")
	activeProviderContact = dto_discovery.Contact{}
	activeProposal        = dto_discovery.ServiceProposal{
		ProviderID:       activeProviderID.Address,
		ProviderContacts: []dto_discovery.Contact{activeProviderContact},
	}
)

func (tc *testContext) TestProposalUnregisteredWhenPingerClosed() {
	stopPinger := make(chan int)
	fakeDiscoveryClient := server.NewClientFake()
	fakeDiscoveryClient.RegisterProposal(activeProposal, nil)

	go PingProposalLoop(activeProposal, fakeDiscoveryClient, nil, stopPinger)

	close(stopPinger) //causes proposal to be unregistered

	proposals, err := fakeDiscoveryClient.FindProposals(activeProposal.ProviderID)

	assert.NoError(tc.T(), err)
	assert.Empty(tc.T(), proposals)
}