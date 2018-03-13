package e2e

import (
	"flag"
	"github.com/mysterium/node/tequilapi/client"
	"github.com/stretchr/testify/assert"
	"testing"
)

var host = flag.String("tequila.host", "localhost", "Specify tequila host for e2e tests")
var port = flag.Int("tequila.port", 4050, "Specify tequila port for e2e tests")

func TestClientStatusIsNotConnectedOnStartup(t *testing.T) {
	//we cannot move tequilApi as var outside - host and port are not initialized yet :(
	//something related to how "go test" calls flag.Parse()
	tequilApi := client.NewClient(*host, *port)

	status, err := tequilApi.Status()
	assert.NoError(t, err)
	assert.Equal(t, "NotConnected", status.Status)
}
