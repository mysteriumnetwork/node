package nats

import (
	dto_discovery "github.com/mysterium/node/service_discovery/dto"
	"github.com/nats-io/go-nats"
	"time"
	"github.com/mysterium/node/ipify"
	"log"
)

func NewContact(identity dto_discovery.Identity) dto_discovery.Contact {
	return dto_discovery.Contact{
		Type: CONTACT_NATS_V1,
		Definition: ContactNATSV1{
			Topic: string(identity),
		},
	}
}

func NewServer(identity dto_discovery.Identity, ipifyClient ipify.Client) *serverNats {
	natsServerIp, err := ipifyClient.GetOutboundIP()

	if err != nil {
		log.Fatal(err)
	}

	return &serverNats{
		myTopic:        string(identity),
		options:        getDefaultOptions(natsServerIp),
		timeoutRequest: 500 * time.Millisecond,
	}
}

func NewClient(identity dto_discovery.Identity) *clientNats {
	// TODO: get natsServerIp for client from discovery service
	return &clientNats{
		myTopic:        string(identity),
		// options:        getDefaultOptions("172.17.0.2"),
		options:        getDefaultOptions("127.0.0.1"),
		timeoutRequest: 500 * time.Millisecond,
	}
}

func getDefaultOptions(natsServerIp string) nats.Options {
	options := nats.GetDefaultOptions()

	options.Servers = []string{
		"nats://" +natsServerIp+":4222",
	}
	return options
}
