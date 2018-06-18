package openvpn

import (
	"github.com/mysterium/node/openvpn/config"
	"github.com/mysterium/node/openvpn/management"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestOpenvpnProcessStartsAndStopsSuccessfully(t *testing.T) {
	process := newOpenvpnProcess("testdata/openvpn-mock-client.sh")
	err := process.Start()
	assert.NoError(t, err)
	time.Sleep(200 * time.Millisecond)
	err = process.Stop()
	assert.NoError(t, err)
}

func TestOpenvpnProcessStartReportsErrorIfCmdWrapperDiesTooEarly(t *testing.T) {
	process := newOpenvpnProcess("testdata/failing-openvpn-mock-client.sh")
	err := process.Start()
	assert.Error(t, err)
}

func newOpenvpnProcess(testExecutablePath string) *openvpnProcess {
	openvpnConfig := &config.GenericConfig{}
	return &openvpnProcess{
		config:     openvpnConfig,
		management: management.NewManagement(management.LocalhostOnRandomPort, "[openvpn-process] "),
		cmd:        NewCmdWrapper(testExecutablePath, "[mock-client] "),
	}
}
