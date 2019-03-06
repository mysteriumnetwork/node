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
	"errors"
	"flag"
	"fmt"
	"log"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/chzyer/readline"
	"github.com/mysteriumnetwork/node/cmd"
	"github.com/mysteriumnetwork/node/core/service"
	"github.com/mysteriumnetwork/node/metadata"
	"github.com/mysteriumnetwork/node/services/noop"
	"github.com/mysteriumnetwork/node/services/openvpn"
	openvpn_service "github.com/mysteriumnetwork/node/services/openvpn/service"
	"github.com/mysteriumnetwork/node/services/wireguard"
	wireguard_service "github.com/mysteriumnetwork/node/services/wireguard/service"
	tequilapi_client "github.com/mysteriumnetwork/node/tequilapi/client"
	"github.com/mysteriumnetwork/node/utils"
	"github.com/urfave/cli"
)

const cliCommandName = "cli"

const serviceHelp = `service <action> [args]
	start	<ProviderID> <ServiceType> [options]
	stop	<ServiceID>
	list
	status	<ServiceID>

	example: service start 0x7d5ee3557775aed0b85d691b036769c17349db23 openvpn --openvpn.port=1194 --openvpn.proto=UDP`

// NewCommand constructs CLI based Mysterium UI with possibility to control quiting
func NewCommand() *cli.Command {
	return &cli.Command{
		Name:  cliCommandName,
		Usage: "Starts a CLI client with a Tequilapi",
		Action: func(ctx *cli.Context) error {
			nodeOptions := cmd.ParseFlagsNode(ctx)
			cmdCLI := &cliApp{
				historyFile: filepath.Join(nodeOptions.Directories.Data, ".cli_history"),
				tequilapi:   tequilapi_client.NewClient(nodeOptions.TequilapiAddress, nodeOptions.TequilapiPort),
			}
			cmd.RegisterSignalCallback(utils.SoftKiller(cmdCLI.Kill))

			return cmdCLI.Run()
		},
	}
}

// cliApp describes CLI based Mysterium UI
type cliApp struct {
	historyFile      string
	tequilapi        *tequilapi_client.Client
	fetchedProposals []tequilapi_client.ProposalDTO
	completer        *readline.PrefixCompleter
	reader           *readline.Instance
}

const redColor = "\033[31m%s\033[0m"
const identityDefaultPassphrase = ""
const statusConnected = "Connected"

var versionSummary = metadata.VersionAsSummary(metadata.LicenseCopyright(
	"type 'license --warranty'",
	"type 'license --conditions'",
))

// Run runs CLI interface synchronously, in the same thread while blocking it
func (c *cliApp) Run() (err error) {
	fmt.Println(versionSummary)

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
		if err == readline.ErrInterrupt && len(line) > 0 {
			continue
		} else if err != nil {
			c.quit()
			return err
		}

		c.handleActions(line)
	}
}

// Kill stops cli
func (c *cliApp) Kill() error {
	c.reader.Clean()
	return c.reader.Close()
}

func (c *cliApp) handleActions(line string) {
	line = strings.TrimSpace(line)

	staticCmds := []struct {
		command string
		handler func()
	}{
		{"exit", c.quit},
		{"quit", c.quit},
		{"help", c.help},
		{"status", c.status},
		{"healthcheck", c.healthcheck},
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
		{command: "version", handler: c.version},
		{command: "license", handler: c.license},
		{command: "registration", handler: c.registration},
		{command: "proposals", handler: c.proposals},
		{command: "service", handler: c.service},
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

func (c *cliApp) service(argsString string) {
	args := strings.Fields(argsString)
	if len(args) == 0 {
		fmt.Println(serviceHelp)
		return
	}

	action := args[0]
	switch action {
	case "start":
		if len(args) < 3 {
			fmt.Println(serviceHelp)
			return
		}
		c.serviceStart(args[1], args[2], args[3:]...)
	case "stop":
		if len(args) < 2 {
			fmt.Println(serviceHelp)
			return
		}
		c.serviceStop(args[1])
	case "status":
		if len(args) < 2 {
			fmt.Println(serviceHelp)
			return
		}
		c.serviceGet(args[1])
	case "list":
		c.serviceList()
	default:
		info(fmt.Sprintf("Unknown action provided: %s", action))
		fmt.Println(serviceHelp)
	}
}

func (c *cliApp) serviceStart(providerID, serviceType string, args ...string) {
	opts, err := parseServiceOptions(serviceType, args...)
	if err != nil {
		info("Failed to parse service options:", err)
		return
	}

	service, err := c.tequilapi.ServiceStart(providerID, serviceType, opts)
	if err != nil {
		info("Failed to start service: ", err)
		return
	}

	status(service.Status,
		"ID: "+service.ID,
		"ProviderID: "+service.Proposal.ProviderID,
		"Type: "+service.Proposal.ServiceType)
}

func (c *cliApp) serviceStop(id string) {
	if err := c.tequilapi.ServiceStop(id); err != nil {
		info("Failed to stop service: ", err)
		return
	}

	status("Stopping", "ID: "+id)
}

func (c *cliApp) serviceList() {
	services, err := c.tequilapi.Services()
	if err != nil {
		info("Failed to get a list of services: ", err)
		return
	}

	for _, service := range services {
		status(service.Status,
			"ID: "+service.ID,
			"ProviderID: "+service.Proposal.ProviderID,
			"Type: "+service.Proposal.ServiceType)
	}
}

func (c *cliApp) serviceGet(id string) {
	service, err := c.tequilapi.Service(id)
	if err != nil {
		info("Failed to get service info: ", err)
		return
	}

	status(service.Status,
		"ID: "+service.ID,
		"ProviderID: "+service.Proposal.ProviderID,
		"Type: "+service.Proposal.ServiceType)
}

func (c *cliApp) connect(argsString string) {
	args := strings.Fields(argsString)

	if len(args) < 3 {
		info("Please type in the provider identity. Connect <consumer-identity> <provider-identity> <service-type> [disable-kill-switch]")
		return
	}

	consumerID, providerID, serviceType := args[0], args[1], args[2]

	var disableKill bool
	var err error
	if len(args) > 3 {
		disableKillStr := args[3]
		disableKill, err = strconv.ParseBool(disableKillStr)
		if err != nil {
			info("Please use true / false for <disable-kill-switch>")
			return
		}
	}

	connectOptions := tequilapi_client.ConnectOptions{DisableKillSwitch: disableKill}

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

	_, err = c.tequilapi.Connect(consumerID, providerID, serviceType, connectOptions)
	if err != nil {
		warn(err)
		return
	}

	success("Connected.")
}

func (c *cliApp) unlock(argsString string) {
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

func (c *cliApp) disconnect() {
	err := c.tequilapi.Disconnect()
	if err != nil {
		warn(err)
		return
	}

	success("Disconnected.")
}

func (c *cliApp) status() {
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
			info("Proposal:", status.Proposal)
			info(fmt.Sprintf("Connection duration: %ds", statistics.Duration))
			info("Bytes sent:", statistics.BytesSent)
			info("Bytes received:", statistics.BytesReceived)
		}
	}
}

func (c *cliApp) healthcheck() {
	healthcheck, err := c.tequilapi.Healthcheck()
	if err != nil {
		warn(err)
		return
	}

	info(fmt.Sprintf("Uptime: %v", healthcheck.Uptime))
	info(fmt.Sprintf("Process: %v", healthcheck.Process))
	info(fmt.Sprintf("Version: %v", healthcheck.Version))
	buildString := metadata.FormatString(healthcheck.BuildInfo.Commit, healthcheck.BuildInfo.Branch, healthcheck.BuildInfo.BuildNumber)
	info(buildString)
}

func (c *cliApp) proposals(filter string) {
	proposals := c.fetchProposals()
	c.fetchedProposals = proposals

	filterMsg := ""
	if filter != "" {
		filterMsg = fmt.Sprintf("(filter: '%s')", filter)
	}
	info(fmt.Sprintf("Found %v proposals %s", len(proposals), filterMsg))

	for _, proposal := range proposals {
		country := proposal.ServiceDefinition.LocationOriginate.Country
		if country == "" {
			country = "Unknown"
		}

		msg := fmt.Sprintf("- provider id: %v, proposal id: %v, country: %v", proposal.ProviderID, proposal.ID, country)

		if filter == "" ||
			strings.Contains(proposal.ProviderID, filter) ||
			strings.Contains(country, filter) {

			info(msg)
		}
	}
}

func (c *cliApp) fetchProposals() []tequilapi_client.ProposalDTO {
	proposals, err := c.tequilapi.Proposals()
	if err != nil {
		warn(err)
		return []tequilapi_client.ProposalDTO{}
	}
	return proposals
}

func (c *cliApp) ip() {
	ip, err := c.tequilapi.GetIP()
	if err != nil {
		warn(err)
		return
	}

	info("IP:", ip)
}

func (c *cliApp) help() {
	info("Mysterium CLI tequilapi commands:")
	fmt.Println(c.completer.Tree("  "))
}

// quit stops cli and client commands and exits application
func (c *cliApp) quit() {
	stop := utils.SoftKiller(c.Kill)
	stop()
}

func (c *cliApp) identities(argsString string) {
	const usage = "identities command:\n    list\n    new [passphrase]"
	if len(argsString) == 0 {
		info(usage)
		return
	}

	switch argsString {
	case "new", "list": // Known sub-commands.
	default:
		warnf("Unknown sub-command '%s'\n", argsString)
		fmt.Println(usage)
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
			fmt.Println("Error occurred:", err)
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

func (c *cliApp) registration(argsString string) {
	if argsString == "" {
		warn("Please supply identity")
		return
	}
	status, err := c.tequilapi.IdentityRegistrationStatus(argsString)
	if err != nil {
		warn("Something went wrong: ", err)
		return
	}
	if status.Registered {
		info("Already registered")
		return
	}
	info("Identity is not registered yet. In order to do that - please call payments contract with the following data")
	info("Public key: part1 ->", status.PublicKey.Part1)
	info("            part2 ->", status.PublicKey.Part2)
	info("Signature: S ->", status.Signature.S)
	info("           R ->", status.Signature.R)
	info("           V ->", status.Signature.V)
	info("OR proceed with direct link:")
	infof(" https://wallet.mysterium.network/?part1=%s&part2=%s&s=%s&r=%s&v=%d\n",
		status.PublicKey.Part1,
		status.PublicKey.Part2,
		status.Signature.S,
		status.Signature.R,
		status.Signature.V)
}

func (c *cliApp) stopClient() {
	err := c.tequilapi.Stop()
	if err != nil {
		warn("Cannot stop client:", err)
	}
	success("Client stopped")
}

func (c *cliApp) version(argsString string) {
	fmt.Println(versionSummary)
}

func (c *cliApp) license(argsString string) {
	if argsString == "warranty" {
		fmt.Print(metadata.LicenseWarranty)
	} else if argsString == "conditions" {
		fmt.Print(metadata.LicenseConditions)
	} else {
		info("identities command:\n    warranty\n    conditions")
	}
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
			"service",
			readline.PcItem("start", readline.PcItemDynamic(
				getIdentityOptionList(tequilapi),
				readline.PcItem("noop"),
				readline.PcItem("openvpn"),
				readline.PcItem("wireguard"),
			)),
			readline.PcItem("stop"),
			readline.PcItem("list"),
			readline.PcItem("status"),
		),
		readline.PcItem(
			"identities",
			readline.PcItem("new"),
			readline.PcItem("list"),
		),
		readline.PcItem("status"),
		readline.PcItem("healthcheck"),
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
		readline.PcItem(
			"license",
			readline.PcItem("warranty"),
			readline.PcItem("conditions"),
		),
		readline.PcItem(
			"registration",
			readline.PcItemDynamic(
				getIdentityOptionList(tequilapi),
			),
		),
	)
}

func parseServiceOptions(serviceType string, args ...string) (service.Options, error) {
	var flags []cli.Flag
	openvpn_service.RegisterFlags(&flags)
	wireguard_service.RegisterFlags(&flags)

	set := flag.NewFlagSet("", flag.ContinueOnError)
	for _, f := range flags {
		f.Apply(set)
	}

	if err := set.Parse(args); err != nil {
		return nil, err
	}

	ctx := cli.NewContext(nil, set, nil)

	switch serviceType {
	case noop.ServiceType:
		return noop.ParseFlags(ctx), nil
	case wireguard.ServiceType:
		return wireguard_service.ParseFlags(ctx), nil
	case openvpn.ServiceType:
		return openvpn_service.ParseFlags(ctx), nil
	}

	return nil, errors.New("service type not found")
}
