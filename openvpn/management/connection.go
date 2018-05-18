/*
 * Copyright (C) 2018 The "MysteriumNetwork/node" Authors.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

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
