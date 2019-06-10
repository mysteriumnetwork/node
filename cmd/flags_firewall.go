package cmd

import (
	"github.com/mysteriumnetwork/node/core/node"
	"github.com/urfave/cli"
)

var (
	enableKillSwitch = cli.BoolFlag{
		Name:  "firewall.killSwitch",
		Usage: "Enable consumer outgoing non tunneled traffic blocking during connections",
	}
	alwaysBlock = cli.BoolFlag{
		Name:  "firewall.killSwitch.always",
		Usage: "Always block non-tunneled outgoing consumer traffic",
	}
)

func RegisterFirewallFlags(flags *[]cli.Flag) {
	*flags = append(*flags, enableKillSwitch, alwaysBlock)
}

func ParseFirewallFlags(ctx *cli.Context) node.OptionsFirewall {
	return node.OptionsFirewall{
		EnableKillSwitch: ctx.GlobalBool(enableKillSwitch.Name),
		BlockAlways:      ctx.GlobalBool(alwaysBlock.Name),
	}
}
