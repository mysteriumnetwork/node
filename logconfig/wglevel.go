package logconfig

import "github.com/rs/zerolog"

// WireGuardLogLevel maps the node's configured zerolog level to a WireGuard
// device log level (0=silent, 1=error, 2=verbose).
// This ensures WireGuard logging respects the --log-level flag.
func WireGuardLogLevel() int {
	switch CurrentLogOptions.LogLevel {
	case zerolog.TraceLevel, zerolog.DebugLevel:
		return 2 // device.LogLevelVerbose
	case zerolog.InfoLevel, zerolog.WarnLevel:
		return 1 // device.LogLevelError
	default:
		return 0 // device.LogLevelSilent
	}
}
