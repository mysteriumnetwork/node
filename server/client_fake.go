package server

import (
	"github.com/mysterium/node/server/dto"

	log "github.com/cihub/seelog"
	"fmt"
)

func NewClientFake() Client {
	return &clientFake{
		connectionConfigByNode: make(map[string]string, 0),
	}
}

type clientFake struct {
	connectionConfigByNode map[string]string
}

func (client *clientFake) NodeRegister(nodeKey, connectionConfig string) (err error) {
	client.connectionConfigByNode[nodeKey] = connectionConfig
	log.Info(MYSTERIUM_API_LOG_PREFIX, "Fake node registered: ", nodeKey)

	return nil
}

func (client *clientFake) SessionCreate(nodeKey string) (session dto.Session, err error) {
	if connectionConfig, ok := client.connectionConfigByNode[nodeKey]; ok {
		session = dto.Session{
			Id: nodeKey + "-session",
			ConnectionConfig: connectionConfig,
		}
		return
	}

	err = fmt.Errorf("Fake node not found: %s", nodeKey)
	return
}