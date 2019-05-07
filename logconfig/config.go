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

package logconfig

import (
	"bytes"
	"text/template"

	"github.com/cihub/seelog"
	"github.com/mysteriumnetwork/node/metadata"
)

const seewayLogXMLConfigTemplate = `
<seelog minlevel="{{.LogLevel}}">
	<outputs formatid="main">
		<console/>
	</outputs>
	<formats>
		<format id="main" format="%UTCDate(2006-01-02T15:04:05.999999999) [%Level] %Msg%n"/>
	</formats>
</seelog>
`

type configParams struct {
	LogLevel string
}

func (cp configParams) String() string {
	tmpl := template.Must(template.New("seelogcfg").Parse(seewayLogXMLConfigTemplate))

	var tpl bytes.Buffer
	err := tmpl.Execute(&tpl, cp)
	if err != nil {
		panic(err)
	}

	return tpl.String()
}

var cfg configParams

func init() {
	cfg = configParams{
		LogLevel: "info",
	}
	if metadata.VersionAsString() == "source.dev-build" {
		cfg.LogLevel = "trace"
	}
}

// Bootstrap loads seelog package into the overall system
func Bootstrap() {
	newLogger, err := seelog.LoggerFromConfigAsString(cfg.String())
	if err != nil {
		seelog.Warn("Error parsing seelog configuration", err)
		return
	}
	err = seelog.UseLogger(newLogger)
	if err != nil {
		seelog.Warn("Error setting new logger for seelog", err)
	}
	seelog.Info("LOG LEVEL: ", cfg.LogLevel)
}
