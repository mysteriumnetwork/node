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

	log "github.com/cihub/seelog"
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

func buildXmlConfig(opts options) string {
	tmpl := template.Must(template.New("seelogcfg").Parse(seewayLogXMLConfigTemplate))

	var tpl bytes.Buffer
	err := tmpl.Execute(&tpl, opts)
	if err != nil {
		panic(err)
	}

	return tpl.String()
}

// Bootstrap loads log package into the overall system
func Bootstrap() {
	newLogger, err := log.LoggerFromConfigAsString(buildXmlConfig(opts))
	if err != nil {
		log.Warn("error parsing log configuration", err)
		return
	}
	err = log.UseLogger(newLogger)
	if err != nil {
		log.Warn("error setting new logger for log", err)
	}
	log.Infof("log level: %s", opts.LogLevel)
}
