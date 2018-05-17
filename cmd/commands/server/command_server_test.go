package server

import (
	"github.com/mysterium/node/server"
	dto_discovery "github.com/mysterium/node/service_discovery/dto"
	"github.com/stretchr/testify/assert"
	"sync"
	"testing"
	"time"
)

var (
	activeProposal = dto_discovery.ServiceProposal{}
)

func TestProposalUnregisteredWhenPingerClosed(t *testing.T) {
	stopPinger := make(chan int)
	fakeDiscoveryClient := server.NewClientFake()
	fakeDiscoveryClient.RegisterProposal(activeProposal, nil)

	finished := make(chan bool)
	fakeCmd := Command{WaitUnregister: &sync.WaitGroup{}}
	fakeCmd.WaitUnregister.Add(1)

	go func() {
		fakeCmd.pingProposalLoop(activeProposal, fakeDiscoveryClient, nil, stopPinger)
		finished <- true
	}()

	close(stopPinger) //causes proposal to be unregistered

	select {
	case _ = <-finished:
		proposals, err := fakeDiscoveryClient.FindProposals(activeProposal.ProviderID)

		assert.NoError(t, err)
		assert.Len(t, proposals, 0)
	case <-time.After(500 * time.Millisecond):
		assert.Fail(t, "failed to stop pinger")
	}
}
