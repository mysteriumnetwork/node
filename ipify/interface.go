package ipify

type Client interface {
	GetIp() (string, error)
}