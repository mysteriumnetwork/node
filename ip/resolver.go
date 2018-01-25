package ip

type Resolver interface {
	GetPublicIP() (string, error)
	GetOutboundIP() (string, error)
}
