package tls

import (
	"github.com/mysterium/node/identity"
	"github.com/mysterium/node/service_discovery/dto"
	"github.com/stretchr/testify/assert"
	"testing"
)

const fakeRunDir = "testdataoutput"

func TestCertificatesAreGenerated(t *testing.T) {
	_, err := NewTLSPrimitives(dto.Location{}, identity.FromAddress("0xdeadbeef"))
	assert.NoError(t, err)
}
