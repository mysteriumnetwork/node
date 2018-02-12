package management

// Management packages contains all functionality related to openvpn management interface
// See https://openvpn.net/index.php/open-source/documentation/miscellaneous/79-management-interface.html

// Middleware used to control openvpn process through management interface
type Middleware interface {
	Start(Connection) error
	Stop(Connection) error
	ConsumeLine(line string) (consumed bool, err error)
}
