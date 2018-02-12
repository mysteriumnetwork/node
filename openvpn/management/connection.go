package management

import (
	"errors"
	"fmt"
	"io"
	"net/textproto"
	"strings"
)

// Connection represents openvpn management interface abstraction for middlewares to be able to send commands to openvpn process
type Connection interface {
	SingleOutputCommand(template string, args ...interface{}) (string, error)
	MultiOutputCommand(template string, args ...interface{}) (string, []string, error)
}

const cmdSuccess = "SUCCESS"
const cmdError = "ERROR"
const endOfCmdOutput = "END"

type socketConnection struct {
	cmdWriter io.Writer
	cmdOutput chan string
}

func newSocketConnection(cmdWriter io.Writer, cmdOutput chan string) *socketConnection {
	return &socketConnection{
		cmdWriter: cmdWriter,
		cmdOutput: cmdOutput,
	}
}

func (sc *socketConnection) SingleOutputCommand(template string, args ...interface{}) (string, error) {
	cmd := fmt.Sprintf(template, args...)

	_, err := fmt.Fprintf(sc.cmdWriter, "%s\n", cmd)
	if err != nil {
		return "", err
	}

	cmdOutput := <-sc.cmdOutput
	outputParts := strings.Split(cmdOutput, ":")
	messageType := textproto.TrimString(outputParts[0])
	messageText := ""
	if len(outputParts) > 1 {
		messageText = textproto.TrimString(outputParts[1])
	}
	switch messageType {
	case cmdSuccess:
		return messageText, nil
	case cmdError:
		return "", errors.New("command error: " + messageText)
	default:
		return "", errors.New("unknown command response: " + cmdOutput)
	}
}

func (sc *socketConnection) MultiOutputCommand(template string, args ...interface{}) (string, []string, error) {
	success, err := sc.SingleOutputCommand(template, args...)
	if err != nil {
		return "", nil, err
	}
	var outputLines []string
	for outputLine := range sc.cmdOutput {
		if outputLine == endOfCmdOutput {
			break
		}
		outputLines = append(outputLines, outputLine)
	}
	return success, outputLines, nil
}
