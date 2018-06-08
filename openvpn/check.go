/*
 * Copyright (C) 2017 The "MysteriumNetwork/node" Authors.
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

package openvpn

import (
	"bufio"
	"bytes"
	"errors"
	log "github.com/cihub/seelog"
	"net/textproto"
	"os/exec"
	"strconv"
	"syscall"
)

// CheckOpenvpnBinary function checks that openvpn is available, given path to openvpn binary
func CheckOpenvpnBinary(openvpnBinary string) error {
	command := exec.Command(openvpnBinary, "--version")
	outputBuffer, cmdResult := command.Output()
	exitCode, err := extractExitCodeFromCmdResult(cmdResult)
	if err != nil {
		return err
	}
	//openvpn returns exit code 1 in case of --version parameter, if anything else is returned - treat as error
	if exitCode != 1 {
		return errors.New("unexpected openvpn code: " + strconv.Itoa(exitCode))
	}

	stringReader := textproto.NewReader(bufio.NewReader(bytes.NewReader(outputBuffer)))
	//openvpn --version produces 5 (and optional 6th) strings as output
	for i := 0; i < 5; i++ {
		str, err := stringReader.ReadLine()
		if err != nil {
			return err
		}
		if str == "" {
			return errors.New("expected more output")
		}
		log.Info("[Openvpn check] ", str)
	}

	//optional custom tag
	str, err := stringReader.ReadLine()
	if err != nil {
		return err
	}
	if str != "" {
		log.Info("[Openvpn check] ", "Custom build: ", str)
	}
	log.Flush()
	return nil
}

//given error value from cmd.Wait() extract exit code if possible, otherwise returns error itself
//this is ugly but there is no standart and os independent way to extract exit status in golang
func extractExitCodeFromCmdResult(cmdResult error) (int, error) {
	exitError, ok := cmdResult.(*exec.ExitError)
	if !ok {
		return 0, cmdResult
	}

	exitStatus, ok := exitError.Sys().(syscall.WaitStatus)
	if !ok {
		return 0, cmdResult
	}
	return exitStatus.ExitStatus(), nil
}
