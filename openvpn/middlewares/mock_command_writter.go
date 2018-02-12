package middlewares

import (
	"fmt"
)

// MockCommandWriter is mock openvpn management interface used for middleware testing
type MockCommandWriter struct {
	WrittenLines []string
	LastLine     string
}

func (conn *MockCommandWriter) PrintfLine(format string, args ...interface{}) error {
	conn.LastLine = fmt.Sprintf(format, args...)
	conn.WrittenLines = append(conn.WrittenLines, conn.LastLine)
	return nil
}
