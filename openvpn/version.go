package openvpn

import (
	"os/exec"
	"syscall"
)

//Version function checks that openvpn is available, given path to openvpn binary
func Version(openvpnBinary string) error {

	process := NewProcess(openvpnBinary, "[openvpn binary check]")
	if err := process.Start([]string{"--version"}); err != nil {
		return err
	}
	cmdResult := process.Wait()

	exitCode, err := extractExitCodeFromCmdResult(cmdResult)
	//openvpn returns exit code 1 in case of --version paramter, if anything else is returned - treat as error
	if err != nil || exitCode != 1 {
		return err
	}
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
