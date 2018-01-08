package nats_discovery

const CONTACT_NATS_V1 = "nats/v1"

type ContactNATSV1 struct {
	// Topic on which client is getting message
	Topic string `json:"topic"`
	// NATS servers used by node and should be contacted via
	BrokerAddresses []string `json:"broker_addresses"`
}
