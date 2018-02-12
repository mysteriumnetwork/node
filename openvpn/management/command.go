package management

// CommandWriter represents command write abstraction for middlewares to be able to send commands to openvpn management interface
type CommandWriter interface {
	PrintfLine(format string, args ...interface{}) error
}
