package ipify

import (
	log "github.com/cihub/seelog"
)

func NewClientFake(IP string) Client {
	return &clientFake{
		ip: IP,
	}
}

type clientFake struct {
	ip string
}

func (client *clientFake) GetIp() (string, error) {
	log.Info(IPIFY_API_LOG_PREFIX, "IP faked: ", client.ip)
	return client.ip, nil
}
