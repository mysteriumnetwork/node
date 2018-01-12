package discovery

// TypeContactNATSV1 defines V1 format for NATS contact
const TypeContactNATSV1 = "nats/v1"

// ContactNATSV1 is definition of NATS contact
type ContactNATSV1 struct {
	// Topic on which client is getting message
	Topic string `json:"topic"`
	// NATS servers used by node and should be contacted via
	BrokerAddresses []string `json:"broker_addresses"`
}
