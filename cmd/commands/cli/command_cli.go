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

package cli

import (
	"fmt"
	"github.com/chzyer/readline"
	"github.com/mysterium/node/cmd"
	tequilapi_client "github.com/mysterium/node/tequilapi/client"
	"io"
	"log"
	"strings"
)

// NewCommand constructs CLI based with possibility to control quiting
func NewCommand(
	historyFile string,
	tequilapi *tequilapi_client.Client,
) *Command {
	return &Command{
		historyFile: historyFile,
		tequilapi:   tequilapi,
	}
}

// Command describes CLI based Mysterium UI
type Command struct {
	historyFile      string
	tequilapi        *tequilapi_client.Client
	fetchedProposals []tequilapi_client.ProposalDTO
	completer        *readline.PrefixCompleter
	reader           *readline.Instance
}

const redColor = "\033[31m%s\033[0m"
const identityDefaultPassphrase = ""
const statusConnected = "Connected"

const license = `Mysterium Node Copyright (C) 2017 The "MysteriumNetwork/node" Authors.
This program comes with ABSOLUTELY NO WARRANTY; for details type ` + "`show w" + `'.
This is free software, and you are welcome to redistribute it
under certain conditions; type ` + "`show c'" + ` for details.
`

// Run runs CLI interface synchronously, in the same thread while blocking it
func (c *Command) Run() (err error) {
	fmt.Print(license + "\n")
	c.fetchedProposals = c.fetchProposals()
	c.completer = newAutocompleter(c.tequilapi, c.fetchedProposals)

	c.reader, err = readline.NewEx(&readline.Config{
		Prompt:          fmt.Sprintf(redColor, "» "),
		HistoryFile:     c.historyFile,
		AutoComplete:    c.completer,
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
	})
	if err != nil {
		return err
	}
	// TODO Should overtake output of CommandRun
	log.SetOutput(c.reader.Stderr())

	for {
		line, err := c.reader.Readline()
		if err == readline.ErrInterrupt {
			if len(line) == 0 {
				c.quit()
			} else {
				continue
			}
		} else if err == io.EOF {
			c.quit()
		}

		c.handleActions(line)
	}
}

// Kill stops cli
func (c *Command) Kill() error {
	c.reader.Clean()
	return c.reader.Close()
}

func (c *Command) handleActions(line string) {
	line = strings.TrimSpace(line)

	staticCmds := []struct {
		command string
		handler func()
	}{
		{"exit", c.quit},
		{"quit", c.quit},
		{"help", c.help},
		{"status", c.status},
		{"proposals", c.proposals},
		{"ip", c.ip},
		{"disconnect", c.disconnect},
		{"stop", c.stopClient},
	}

	argCmds := []struct {
		command string
		handler func(argsString string)
	}{
		{command: "connect", handler: c.connect},
		{command: "unlock", handler: c.unlock},
		{command: "identities", handler: c.identities},
	}

	for _, cmd := range staticCmds {
		if line == cmd.command {
			cmd.handler()
			return
		}
	}

	for _, cmd := range argCmds {
		if strings.HasPrefix(line, cmd.command) {
			argsString := strings.TrimSpace(line[len(cmd.command):])
			cmd.handler(argsString)
			return
		}
	}

	if len(line) > 0 {
		c.help()
	}
}

func (c *Command) connect(argsString string) {
	if len(argsString) == 0 {
		info("Press tab to select identity or create a new one. Connect <consumer-identity> <provider-identity>")
		return
	}

	identities := strings.Fields(argsString)

	if len(identities) != 2 {
		info("Please type in the provider identity. Connect <consumer-identity> <provider-identity>")
		return
	}

	consumerID, providerID := identities[0], identities[1]

	if consumerID == "new" {
		id, err := c.tequilapi.NewIdentity(identityDefaultPassphrase)
		if err != nil {
			warn(err)
			return
		}
		consumerID = id.Address
		success("New identity created:", consumerID)
	}

	status("CONNECTING", "from:", consumerID, "to:", providerID)

	_, err := c.tequilapi.Connect(consumerID, providerID)
	if err != nil {
		warn(err)
		return
	}

	success("Connected.")
}

func (c *Command) unlock(argsString string) {
	unlockSignature := "Unlock <identity> [passphrase]"
	if len(argsString) == 0 {
		info("Press tab to select identity.", unlockSignature)
		return
	}

	args := strings.Fields(argsString)
	var identity, passphrase string

	if len(args) == 1 {
		identity, passphrase = args[0], ""
	} else if len(args) == 2 {
		identity, passphrase = args[0], args[1]
	} else {
		info("Please type in identity and optional passphrase.", unlockSignature)
		return
	}

	info("Unlocking", identity)
	err := c.tequilapi.Unlock(identity, passphrase)
	if err != nil {
		warn(err)
		return
	}

	success(fmt.Sprintf("Identity %s unlocked.", identity))
}

func (c *Command) disconnect() {
	err := c.tequilapi.Disconnect()
	if err != nil {
		warn(err)
		return
	}

	success("Disconnected.")
}

func (c *Command) status() {
	status, err := c.tequilapi.Status()
	if err != nil {
		warn(err)
	} else {
		info("Status:", status.Status)
		info("SID:", status.SessionID)
	}

	if status.Status == statusConnected {
		statistics, err := c.tequilapi.ConnectionStatistics()
		if err != nil {
			warn(err)
		} else {
			info(fmt.Sprintf("Connection duration: %ds", statistics.Duration))
			info("Bytes sent:", statistics.BytesSent)
			info("Bytes received:", statistics.BytesReceived)
		}
	}
}

func (c *Command) proposals() {
	proposals := c.fetchProposals()
	c.fetchedProposals = proposals
	info(fmt.Sprintf("Found %v proposals", len(proposals)))

	for _, proposal := range proposals {
		country := proposal.ServiceDefinition.LocationOriginate.Country
		if country == "" {
			country = "Unknown"
		}
		msg := fmt.Sprintf("- provider id: %v, proposal id: %v, country: %v", proposal.ProviderID, proposal.ID, country)
		info(msg)
	}
}

func (c *Command) fetchProposals() []tequilapi_client.ProposalDTO {
	proposals, err := c.tequilapi.Proposals()
	if err != nil {
		warn(err)
		return []tequilapi_client.ProposalDTO{}
	}
	return proposals
}

func (c *Command) ip() {
	ip, err := c.tequilapi.GetIP()
	if err != nil {
		warn(err)
		return
	}

	info("IP:", ip)
}

func (c *Command) help() {
	info("Mysterium CLI tequilapi commands:")
	fmt.Println(c.completer.Tree("  "))
}

// quit stops cli and client commands and exits application
func (c *Command) quit() {
	stop := cmd.NewApplicationStopper(c.Kill)
	stop()
}

func (c *Command) identities(argsString string) {
	const usage = "identities command:\n    list\n    new [passphrase]"
	if len(argsString) == 0 {
		info(usage)
		return
	}

	args := strings.Fields(argsString)
	if len(args) < 1 {
		info(usage)
		return
	}

	action := args[0]
	if action == "list" {
		if len(args) > 1 {
			info(usage)
			return
		}
		ids, err := c.tequilapi.GetIdentities()
		if err != nil {
			fmt.Println("Error occured:", err)
			return
		}

		for _, id := range ids {
			status("+", id.Address)
		}
		return
	}

	if action == "new" {
		var passphrase string
		if len(args) == 1 {
			passphrase = identityDefaultPassphrase
		} else if len(args) == 2 {
			passphrase = args[1]
		} else {
			info(usage)
			return
		}

		id, err := c.tequilapi.NewIdentity(passphrase)
		if err != nil {
			warn(err)
			return
		}
		success("New identity created:", id.Address)
	}
}

func (c *Command) stopClient() {
	err := c.tequilapi.Stop()
	if err != nil {
		warn("Cannot stop client:", err)
	}
	success("Client stopped")
}

func getIdentityOptionList(tequilapi *tequilapi_client.Client) func(string) []string {
	return func(line string) []string {
		identities := []string{"new"}
		ids, err := tequilapi.GetIdentities()
		if err != nil {
			warn(err)
			return identities
		}
		for _, id := range ids {
			identities = append(identities, id.Address)
		}

		return identities
	}
}

func getProposalOptionList(proposals []tequilapi_client.ProposalDTO) func(string) []string {
	return func(line string) []string {
		var providerIDS []string
		for _, proposal := range proposals {
			providerIDS = append(providerIDS, proposal.ProviderID)
		}
		return providerIDS
	}
}

func newAutocompleter(tequilapi *tequilapi_client.Client, proposals []tequilapi_client.ProposalDTO) *readline.PrefixCompleter {
	return readline.NewPrefixCompleter(
		readline.PcItem(
			"connect",
			readline.PcItemDynamic(
				getIdentityOptionList(tequilapi),
				readline.PcItemDynamic(
					getProposalOptionList(proposals),
				),
			),
		),
		readline.PcItem(
			"identities",
			readline.PcItem("new"),
			readline.PcItem("list"),
		),
		readline.PcItem("status"),
		readline.PcItem("proposals"),
		readline.PcItem("ip"),
		readline.PcItem("disconnect"),
		readline.PcItem("help"),
		readline.PcItem("quit"),
		readline.PcItem("stop"),
		readline.PcItem(
			"unlock",
			readline.PcItemDynamic(
				getIdentityOptionList(tequilapi),
			),
		),
	)
}
