package logconfig

import "github.com/cihub/seelog"

const seewayLogXmlConfig = `
<seelog>
	<outputs formatid="main">
		<console/>
	</outputs>
	<formats>
		<format id="main" format="%UTCDate(2006-01-02T15:04:05.999999999) [%Level] %Msg%n"/>
	</formats>
</seelog>
`

func init() {
	newLogger, err := seelog.LoggerFromConfigAsString(seewayLogXmlConfig)
	if err != nil {
		seelog.Warn("Error parsing seelog configuration", err)
		return
	}
	err = seelog.UseLogger(newLogger)
	if err != nil {
		seelog.Warn("Error setting new logger for seelog", err)
	}
}
