package server

import (
	"github.com/mysterium/node/server/dto"

	log "github.com/cihub/seelog"
)

func NewClientFake(connectionConfigStatic string) Client {
	return &clientFake{
		connectionConfigStatic: connectionConfigStatic,
	}
}

type clientFake struct {
	connectionConfigStatic string
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

	log.Info(MYSTERIUM_API_LOG_PREFIX, "Created new faked session: ", session.Id)
	session = dto.Session{
		Id: "1",
		ConnectionConfig: client.connectionConfigStatic,
	}
	return
}