package management

// Connection represents openvpn management interface abstraction for middlewares to be able to send commands to openvpn process
type Connection interface {
	PrintfLine(format string, args ...interface{}) error
}
