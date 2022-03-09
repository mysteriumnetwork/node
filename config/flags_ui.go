package config

import "github.com/urfave/cli/v2"

var (
	// FlagFeatureRestart restart feature toggle
	FlagFeatureRestart = cli.StringFlag{
		Name:  "ui.features",
		Usage: "Enable NodeUI features. Multiple features are joined by string (e.g feature1,feature2,...",
		Value: "",
	}
)

// RegisterFlagsUI register Node UI flags to the list
func RegisterFlagsUI(flags *[]cli.Flag) {
	*flags = append(
		*flags,
		&FlagFeatureRestart,
	)
}

// ParseFlagsUI parse Node UI flags
func ParseFlagsUI(ctx *cli.Context) {
	Current.ParseStringFlag(ctx, FlagFeatureRestart)
}
