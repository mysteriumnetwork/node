package management

// Interface represents openvpn management interface abstraction for middlewares to be able to send commands to openvpn process
type Interface interface {
	PrintfLine(format string, args ...interface{}) error
}
