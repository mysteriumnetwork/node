package e2e

import (
	"flag"
	"github.com/cihub/seelog"
	"github.com/mysterium/node/tequilapi/client"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

var host = flag.String("tequila.host", "localhost", "Specify tequila host for e2e tests")
var port = flag.Int("tequila.port", 4050, "Specify tequila port for e2e tests")

func TestClientConnectsToNode(t *testing.T) {
	//we cannot move tequilApi as var outside - host and port are not initialized yet :(
	//something related to how "go test" calls flag.Parse()
	tequilApi := client.NewClient(*host, *port)

	status, err := tequilApi.Status()
	assert.NoError(t, err)
	assert.Equal(t, "NotConnected", status.Status)

	identities, err := tequilApi.GetIdentities()
	assert.NoError(t, err)

	var identity client.IdentityDTO
	if len(identities) < 1 {
		identity, err = tequilApi.NewIdentity("")
		assert.NoError(t, err)
	} else {
		identity = identities[0]
	}
	seelog.Info("Client identity is: ", identity.Address)

	err = tequilApi.Unlock(identity.Address, "")
	assert.NoError(t, err)

	proposals, err := tequilApi.Proposals()
	if err != nil {
		assert.Error(t, err)
		assert.FailNow(t, "Proposals returned error - no point to continue")
	}

	//expect exactly one proposal
	if len(proposals) != 1 {
		assert.FailNow(t, "Exactly one proposal is expected - something is not right!")
	}

	proposal := proposals[0]
	seelog.Info("Connecting to provider: ", proposal)

	_, err = tequilApi.Connect(identity.Address, proposal.ProviderID)
	assert.NoError(t, err)

	time.Sleep(10 * time.Second)
	status, err = tequilApi.Status()
	assert.NoError(t, err)
	assert.Equal(t, "Connected", status.Status)

	err = tequilApi.Disconnect()
	assert.NoError(t, err)

	time.Sleep(10 * time.Second)
	status, err = tequilApi.Status()
	assert.NoError(t, err)
	assert.Equal(t, "NotConnected", status.Status)

}
