package ip

import (
	log "github.com/cihub/seelog"
)

func NewFakeResolver(IP string) Resolver {
	return &fakeResolver{
		ip: IP,
	}
}

type fakeResolver struct {
	ip         string
	outboundIP string
}

func (client *fakeResolver) GetPublicIP() (string, error) {
	log.Info(IpifiAPILogPrefix, "IP faked: ", client.ip)
	return client.ip, nil
}

func (client *fakeResolver) GetOutboundIP() (string, error) {
	log.Info(IpifiAPILogPrefix, "IP faked: ", client.outboundIP)
	return client.outboundIP, nil
}
