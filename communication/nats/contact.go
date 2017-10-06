package nats

const CONTACT_NATS_V1 = "nats/v1"

type ContactNATSV1 struct {
	// Topic on which client is getting message
	Topic string
}
