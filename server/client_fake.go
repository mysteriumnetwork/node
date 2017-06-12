package server

import (
	"github.com/MysteriumNetwork/node/server/dto"

	"fmt"
	log "github.com/cihub/seelog"
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

func (client *clientFake) NodeSendStats(nodeKey string, sessionStats []dto.SessionStats) (err error) {
	log.Info(MYSTERIUM_API_LOG_PREFIX, "Node stats sent: ", nodeKey)

	return nil
}

func (client *clientFake) SessionCreate(nodeKey string) (session dto.Session, err error) {
	if connectionConfig, ok := client.connectionConfigByNode[nodeKey]; ok {
		session = dto.Session{
			Id:               nodeKey + "-session",
			ConnectionConfig: connectionConfig,
		}
		return
	}

	err = fmt.Errorf("Fake node not found: %s", nodeKey)
	return
}

func (client *clientFake) SessionSendStats(sessionId string, sessionStats dto.SessionStats) (err error) {
	log.Info(MYSTERIUM_API_LOG_PREFIX, "Session stats sent: ", sessionId)

	return nil
}
