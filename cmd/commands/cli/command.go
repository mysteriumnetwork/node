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
	"flag"
	"fmt"
	"io"
	stdlog "log"
	"path/filepath"
	"strings"

	"github.com/chzyer/readline"
	"github.com/mysteriumnetwork/node/cmd"
	"github.com/mysteriumnetwork/node/config"
	"github.com/mysteriumnetwork/node/config/urfavecli/clicontext"
	"github.com/mysteriumnetwork/node/core/connection"
	"github.com/mysteriumnetwork/node/core/node"
	"github.com/mysteriumnetwork/node/core/service"
	"github.com/mysteriumnetwork/node/metadata"
	"github.com/mysteriumnetwork/node/services"
	"github.com/mysteriumnetwork/node/services/noop"
	"github.com/mysteriumnetwork/node/services/openvpn"
	openvpn_service "github.com/mysteriumnetwork/node/services/openvpn/service"
	"github.com/mysteriumnetwork/node/services/wireguard"
	wireguard_service "github.com/mysteriumnetwork/node/services/wireguard/service"
	tequilapi_client "github.com/mysteriumnetwork/node/tequilapi/client"
	"github.com/mysteriumnetwork/node/utils"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"gopkg.in/urfave/cli.v1"
)

const cliCommandName = "cli"

const serviceHelp = `service <action> [args]
	start	<ProviderID> <ServiceType> [options]
	stop	<ServiceID>
	status	<ServiceID>
	list
	sessions

	example: service start 0x7d5ee3557775aed0b85d691b036769c17349db23 openvpn --openvpn.port=1194 --openvpn.proto=UDP`

// NewCommand constructs CLI based Mysterium UI with possibility to control quiting
func NewCommand() *cli.Command {
	return &cli.Command{
		Name:   cliCommandName,
		Usage:  "Starts a CLI client with a Tequilapi",
		Before: clicontext.LoadUserConfigQuietly,
		Action: func(ctx *cli.Context) error {
			config.ParseFlagsNode(ctx)
			nodeOptions := node.GetOptions()
			cmdCLI := &cliApp{
				historyFile: filepath.Join(nodeOptions.Directories.Data, ".cli_history"),
				tequilapi:   tequilapi_client.NewClient(nodeOptions.TequilapiAddress, nodeOptions.TequilapiPort),
			}
			cmd.RegisterSignalCallback(utils.SoftKiller(cmdCLI.Kill))

			return describeQuit(cmdCLI.Run())
		},
	}
}

func describeQuit(err error) error {
	if err == nil || err == io.EOF || err == readline.ErrInterrupt {
		log.Info().Msg("Stopping application")
		return nil
	}
	log.Error().Err(err).Stack().Msg("Terminating application due to error")
	return err
}

// cliApp describes CLI based Mysterium UI
type cliApp struct {
	historyFile      string
	tequilapi        *tequilapi_client.Client
	fetchedProposals []tequilapi_client.ProposalDTO
	completer        *readline.PrefixCompleter
	reader           *readline.Instance

	currentConsumerID string
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
	stdlog.SetOutput(c.reader.Stderr())

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
		{"nat", c.natStatus},
		{"location", c.location},
		{"disconnect", c.disconnect},
		{"stop", c.stopClient},
	}

	argCmds := []struct {
		command string
		handler func(argsString string)
	}{
		{"connect", c.connect},
		{"identities", c.identities},
		{"payout", c.payout},
		{"version", c.version},
		{"license", c.license},
		{"proposals", c.proposals},
		{"service", c.service},
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
	case "sessions":
		c.serviceSessions()
	default:
		info(fmt.Sprintf("Unknown action provided: %s", action))
		fmt.Println(serviceHelp)
	}
}

func (c *cliApp) serviceStart(providerID, serviceType string, args ...string) {
	opts, sharedOpts, err := parseStartFlags(serviceType, args...)
	if err != nil {
		info("Failed to parse service options:", err)
		return
	}

	ap := tequilapi_client.AccessPoliciesRequest{
		IDs: sharedOpts.AccessPolicies,
	}

	service, err := c.tequilapi.ServiceStart(providerID, serviceType, opts, ap)
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

func (c *cliApp) serviceSessions() {
	sessions, err := c.tequilapi.ServiceSessions()
	if err != nil {
		info("Failed to get a list of sessions: ", err)
		return
	}

	status("Current sessions", len(sessions.Sessions))
	for _, session := range sessions.Sessions {
		status("ID: "+session.ID, "ConsumerID: "+session.ConsumerID)
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

	helpMsg := "Please type in the provider identity. connect <consumer-identity> <provider-identity> <service-type> [dns=auto|provider|system|1.1.1.1] [disable-kill-switch]"
	if len(args) < 3 {
		info(helpMsg)
		return
	}

	consumerID, providerID, serviceType := args[0], args[1], args[2]

	var disableKillSwitch bool
	var dns connection.DNSOption
	var err error
	for _, arg := range args[3:] {
		if strings.HasPrefix(arg, "dns=") {
			kv := strings.Split(arg, "=")
			dns, err = connection.NewDNSOption(kv[1])
			if err != nil {
				warn("Invalid value: ", err)
				info(helpMsg)
				return
			}
			continue
		}
		switch arg {
		case "disable-kill-switch":
			disableKillSwitch = true
		default:
			warn("Unexpected arg:", arg)
			info(helpMsg)
			return
		}
	}

	connectOptions := tequilapi_client.ConnectOptions{
		DNS:               dns,
		DisableKillSwitch: disableKillSwitch,
	}

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

	accountantID := config.GetString(config.FlagAccountantID)
	_, err = c.tequilapi.ConnectionCreate(consumerID, providerID, accountantID, serviceType, connectOptions)
	if err != nil {
		warn(err)
		return
	}

	c.currentConsumerID = consumerID

	success("Connected.")
}

func (c *cliApp) payout(argsString string) {
	args := strings.Fields(argsString)

	const usage = "payout command:\n    set"
	if len(args) == 0 {
		info(usage)
		return
	}

	action := args[0]
	switch action {
	case "set":
		payoutSignature := "payout set <identity> <ethAddress>"
		if len(args) < 2 {
			info("Please provide identity. You can select one by pressing tab.\n", payoutSignature)
			return
		}

		var identity, ethAddress string
		if len(args) > 2 {
			identity, ethAddress = args[1], args[2]
		} else {
			info("Please type in identity and Ethereum address.\n", payoutSignature)
			return
		}

		err := c.tequilapi.Payout(identity, ethAddress)
		if err != nil {
			warn(err)
			return
		}

		success(fmt.Sprintf("Payout address %s registered.", ethAddress))
	default:
		warnf("Unknown sub-command '%s'\n", action)
		fmt.Println(usage)
		return
	}
}

func (c *cliApp) disconnect() {
	err := c.tequilapi.ConnectionDestroy()
	if err != nil {
		warn(err)
		return
	}
	c.currentConsumerID = ""
	success("Disconnected.")
}

func (c *cliApp) status() {
	status, err := c.tequilapi.ConnectionStatus()
	if err != nil {
		warn(err)
	} else {
		info("Status:", status.Status)
		info("SID:", status.SessionID)
	}

	ip, err := c.tequilapi.ConnectionIP()
	if err != nil {
		warn(err)
	} else {
		info("IP:", ip)
	}

	location, err := c.tequilapi.ConnectionLocation()
	if err != nil {
		warn(err)
	} else {
		info(fmt.Sprintf("Location: %s, %s (%s - %s)", location.City, location.Country, location.UserType, location.ISP))
	}

	if status.Status == statusConnected {
		info("Proposal:", status.Proposal)

		statistics, err := c.tequilapi.ConnectionStatistics()
		if err != nil {
			warn(err)
		} else {
			identityStatus, err := c.tequilapi.GetIdentityStatus(c.currentConsumerID)
			if err != nil {
				warn(err)
			}
			info("Balance:", identityStatus.Balance)
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

func (c *cliApp) natStatus() {
	status, err := c.tequilapi.NATStatus()
	if err != nil {
		warn("Failed to retrieve NAT traversal status:", err)
		return
	}

	if status.Error == "" {
		infof("NAT traversal status: %q\n", status.Status)
	} else {
		infof("NAT traversal status: %q (error: %q)\n", status.Status, status.Error)
	}
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

		var policies []string
		for _, policy := range proposal.AccessPolicies {
			policies = append(policies, policy.ID)
		}

		msg := fmt.Sprintf("- provider id: %v\ttype: %v\tcountry: %v\taccess policies: %v", proposal.ProviderID, proposal.ServiceType, country, strings.Join(policies, ","))

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

func (c *cliApp) location() {
	location, err := c.tequilapi.OriginLocation()
	if err != nil {
		warn(err)
		return
	}

	info(fmt.Sprintf("Location: %s, %s (%s - %s)", location.City, location.Country, location.UserType, location.ISP))
}

func (c *cliApp) help() {
	info("Mysterium CLI commands:")
	fmt.Println(c.completer.Tree("  "))
}

// quit stops cli and client commands and exits application
func (c *cliApp) quit() {
	stop := utils.SoftKiller(c.Kill)
	stop()
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
		var identities []string
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
	connectOpts := []readline.PrefixCompleterInterface{
		readline.PcItem("dns=auto"),
		readline.PcItem("dns=provider"),
		readline.PcItem("dns=system"),
		readline.PcItem("dns=1.1.1.1"),
	}
	return readline.NewPrefixCompleter(
		readline.PcItem(
			"connect",
			readline.PcItemDynamic(
				getIdentityOptionList(tequilapi),
				readline.PcItemDynamic(
					getProposalOptionList(proposals),
					readline.PcItem("noop", connectOpts...),
					readline.PcItem("openvpn", connectOpts...),
					readline.PcItem("wireguard", connectOpts...),
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
			readline.PcItem("sessions"),
		),
		readline.PcItem(
			"identities",
			readline.PcItem("list"),
			readline.PcItem("new"),
			readline.PcItem("unlock", readline.PcItemDynamic(getIdentityOptionList(tequilapi))),
			readline.PcItem("register", readline.PcItemDynamic(getIdentityOptionList(tequilapi))),
			readline.PcItem("topup", readline.PcItemDynamic(getIdentityOptionList(tequilapi))),
		),
		readline.PcItem("status"),
		readline.PcItem("healthcheck"),
		readline.PcItem("nat"),
		readline.PcItem("proposals"),
		readline.PcItem("location"),
		readline.PcItem("disconnect"),
		readline.PcItem("help"),
		readline.PcItem("quit"),
		readline.PcItem("stop"),
		readline.PcItem(
			"payout",
			readline.PcItem("set", readline.PcItemDynamic(getIdentityOptionList(tequilapi))),
		),
		readline.PcItem(
			"license",
			readline.PcItem("warranty"),
			readline.PcItem("conditions"),
		),
		readline.PcItem("registration", readline.PcItemDynamic(getIdentityOptionList(tequilapi))),
	)
}

func parseStartFlags(serviceType string, args ...string) (service.Options, config.Options, error) {
	var flags []cli.Flag
	config.RegisterFlagsServiceShared(&flags)
	config.RegisterFlagsServiceOpenvpn(&flags)
	config.RegisterFlagsServiceWireguard(&flags)

	set := flag.NewFlagSet("", flag.ContinueOnError)
	for _, f := range flags {
		f.Apply(set)
	}

	if err := set.Parse(args); err != nil {
		return nil, config.Options{}, err
	}

	ctx := cli.NewContext(nil, set, nil)

	config.ParseFlagsServiceShared(ctx)
	switch serviceType {
	case noop.ServiceType:
		return noop.ParseFlags(ctx), services.SharedConfiguredOptions(), nil
	case wireguard.ServiceType:
		config.ParseFlagsServiceWireguard(ctx)
		return wireguard_service.GetOptions(), services.SharedConfiguredOptions(), nil
	case openvpn.ServiceType:
		config.ParseFlagsServiceOpenvpn(ctx)
		return openvpn_service.GetOptions(), services.SharedConfiguredOptions(), nil
	}

	return nil, config.Options{}, errors.New("service type not found")
}
