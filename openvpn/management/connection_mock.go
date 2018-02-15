package management

import (
	"fmt"
)

// MockConnection is mock openvpn management interface used for middleware testing
type MockConnection struct {
	WrittenLines      []string
	LastLine          string
	CommandResult     string
	MultilineResponse []string
}

func (conn *MockConnection) SingleLineCommand(format string, args ...interface{}) (string, error) {
	conn.LastLine = fmt.Sprintf(format, args...)
	conn.WrittenLines = append(conn.WrittenLines, conn.LastLine)
	return conn.CommandResult, nil
}

func (conn *MockConnection) MultiLineCommand(format string, args ...interface{}) (string, []string, error) {
	_, _ = conn.SingleLineCommand(format, args...)
	return conn.CommandResult, conn.MultilineResponse, nil
}
