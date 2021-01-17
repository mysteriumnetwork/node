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
	"bytes"
	"text/template"

	"github.com/rs/zerolog/log"

	"github.com/takama/daemon"
)

const daemonName = "myst_supervisor"
const description = "Myst Supervisor"
const descriptor = `
[Unit]
Description={{.Description}}

[Service]
PIDFile=/run/{{.Name}}.pid
ExecStartPre=/bin/rm -f /run/{{.Name}}.pid
ExecStart={{.Path}} {{.Args}}
Restart=on-failure

[Install]
WantedBy=multi-user.target
`

// Install installs service for linux. Not implemented yet.
func Install(options Options) error {
	if !options.valid() {
		return errInvalid
	}

	log.Info().Msg("Configuring Myst Supervisor")
	dmn, err := mystSupervisorDaemon(options)
	if err != nil {
		return err
	}

	log.Info().Msg("Cleaning up previous installation")
	clean(dmn)

	log.Info().Msg("Installing daemon")
	output, err := dmn.Install()
	if err != nil {
		log.Info().Msg(output)
		return err
	}

	output, err = dmn.Start()
	if err != nil {
		log.Info().Msg(output)
		return err
	}
	return nil
}

func mystSupervisorDaemon(options Options) (daemon.Daemon, error) {
	dmn, err := daemon.New(daemonName, description, daemon.SystemDaemon)
	if err != nil {
		return nil, err
	}

	t, err := template.New("unit-descriptor").Parse(description)
	if err != nil {
		return nil, err
	}
	buffer := new(bytes.Buffer)
	err = t.Execute(buffer, map[string]string{
		"Path":        options.SupervisorPath,
		"Description": "{{.Description}}",
		"Name":        "{{.Name}}",
		"Args":        "{{.Args}}",
	})
	if err != nil {
		return nil, err
	}

	err = dmn.SetTemplate(descriptor)
	if err != nil {
		return nil, err
	}
	return dmn, err
}

func clean(d daemon.Daemon) error {
	output, err := d.Stop()
	if err != nil {
		log.Info().Msgf("%s\t%s", output, err)
	}
	output, err = d.Remove()
	if err != nil {
		log.Info().Msgf("%s\t%s", output, err)
	}
	return err
}

// Uninstall installs service for linux. Not implemented yet.
func Uninstall() error {
	log.Info().Msg("Uninstalling Myst Supervisor daemon")
	dmn, err := daemon.New(daemonName, description, daemon.SystemDaemon)
	if err != nil {
		return err
	}
	return clean(dmn)
}
