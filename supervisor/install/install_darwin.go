/*
 * Copyright (C) 2020 The "MysteriumNetwork/node" Authors.
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

package install

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"text/template"

	"github.com/rs/zerolog/log"
)

const daemonID = "network.mysterium.myst_supervisor"
const plistPath = "/Library/LaunchDaemons/" + daemonID + ".plist"
const plistTpl = `
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>{{.DaemonID}}</string>
	<key>ProgramArguments</key>
	<array>
		<string>{{.SupervisorPath}}</string>
	</array>
	<key>KeepAlive</key>
	<true/>
</dict>
</plist>
`

// Install installs launchd supervisor daemon on Darwin OS.
func Install(options Options) error {
	if !options.valid() {
		return errInvalid
	}

	log.Info().Msg("Cleaning up previous installation")
	clean()

	log.Info().Msg("Installing launchd daemon")
	tpl, err := template.New("plistTpl").Parse(plistTpl)
	if err != nil {
		return fmt.Errorf("could not create template for %s: %w", plistPath, err)
	}
	fd, err := os.Create(plistPath)
	if err != nil {
		return fmt.Errorf("could not create file %s: %w", plistPath, err)
	}
	err = tpl.Execute(fd, map[string]string{
		"DaemonID":       daemonID,
		"SupervisorPath": options.SupervisorPath,
	})
	if err != nil {
		return fmt.Errorf("could not generate %s: %w", plistPath, err)
	}
	out, err := runV("launchctl", "load", plistPath)
	if err == nil && strings.Contains(out, "Invalid property") {
		err = errors.New("invalid plist file")
	}
	if err != nil {
		return fmt.Errorf("could not load launch daemon: %w", err)
	}
	return nil
}

// Uninstall launchd supervisor daemon on Darwin OS.
func Uninstall() error {
	log.Info().Msg("Uninstalling launchd daemon")
	return clean()
}

func clean() error {
	if _, err := runV("launchctl", "unload", plistPath); err != nil {
		return err
	}
	return os.RemoveAll(plistPath)
}

func runV(c ...string) (string, error) {
	cmd := exec.Command(c[0], c[1:]...)
	output, err := cmd.CombinedOutput()
	log.Printf("[%v] out:\n%s", strings.Join(c, " "), output)
	if err != nil {
		return "", err
	}
	return string(output), nil
}
