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
	outboundIp string
}

func (client *clientFake) GetPublicIP() (string, error) {
	log.Info(IPIFY_API_LOG_PREFIX, "IP faked: ", client.ip)
	return client.ip, nil
}

func (client *clientFake) GetOutboundIP() (string, error) {
	log.Info(IPIFY_API_LOG_PREFIX, "IP faked: ", client.outboundIp)
	return client.outboundIp, nil
}
