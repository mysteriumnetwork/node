package management

import (
	"errors"
	"fmt"
	"io"
	"net/textproto"
	"strings"
)

const cmdSuccess = "SUCCESS"
const cmdError = "ERROR"
const endOfCmdOutput = "END"

type channelConnection struct {
	cmdWriter io.Writer
	cmdOutput chan string
}

func newChannelConnection(cmdWriter io.Writer, cmdOutput chan string) *channelConnection {
	return &channelConnection{
		cmdWriter: cmdWriter,
		cmdOutput: cmdOutput,
	}
}

func (sc *channelConnection) SingleLineCommand(template string, args ...interface{}) (string, error) {
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

func (sc *channelConnection) MultiLineCommand(template string, args ...interface{}) (string, []string, error) {
	success, err := sc.SingleLineCommand(template, args...)
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
