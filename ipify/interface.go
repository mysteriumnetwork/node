package ipify

type Client interface {
	GetPublicIP() (string, error)
	GetOutboundIP() (string, error)
}
