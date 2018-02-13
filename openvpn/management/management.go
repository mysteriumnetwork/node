package management

// Management packages contains all functionality related to openvpn management interface
// See https://openvpn.net/index.php/open-source/documentation/miscellaneous/79-management-interface.html

// Connection represents openvpn management interface abstraction for middlewares to be able to send commands to openvpn process
type Connection interface {
	SingleLineCommand(template string, args ...interface{}) (string, error)
	MultiLineCommand(template string, args ...interface{}) (string, []string, error)
}

// Middleware used to control openvpn process through management interface
// It's guaranteed that ConsumeLine callback will be called AFTER Start callback is finished
// Connection passed on Stop callback can be already closed - expect errors when sending commands
// For efficiency and simplicity purposes ConsumeLine for each middleware is called from the same goroutine which
// consumes events from channel - avoid long running operations at all costs
type Middleware interface {
	Start(Connection) error
	Stop(Connection) error
	ConsumeLine(line string) (consumed bool, err error)
}
