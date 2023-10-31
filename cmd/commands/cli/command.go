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
	"io"
	stdlog "log"
	"path/filepath"
	"strings"
	"time"

	"github.com/anmitsu/go-shlex"
	"github.com/chzyer/readline"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"

	"github.com/mysteriumnetwork/node/cmd"
	"github.com/mysteriumnetwork/node/cmd/commands/cli/clio"
	"github.com/mysteriumnetwork/node/config"
	"github.com/mysteriumnetwork/node/config/remote"
	"github.com/mysteriumnetwork/node/core/connection"
	"github.com/mysteriumnetwork/node/core/connection/connectionstate"
	"github.com/mysteriumnetwork/node/datasize"
	"github.com/mysteriumnetwork/node/metadata"
	"github.com/mysteriumnetwork/node/money"
	nattype "github.com/mysteriumnetwork/node/nat"
	"github.com/mysteriumnetwork/node/services"
	tequilapi_client "github.com/mysteriumnetwork/node/tequilapi/client"
	"github.com/mysteriumnetwork/node/tequilapi/contract"
	"github.com/mysteriumnetwork/node/utils"
	"github.com/mysteriumnetwork/terms/terms-go"
)

// CommandName is the name which is used to call this command
const CommandName = "cli"

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
		Name:  CommandName,
		Usage: "Starts a CLI client with a Tequilapi",
		Flags: []cli.Flag{&config.FlagAgreedTermsConditions, &config.FlagTequilapiAddress, &config.FlagTequilapiPort},
		Action: func(ctx *cli.Context) error {
			client, err := clio.NewTequilApiClient(ctx)
			if err != nil {
				return err
			}

			cfg, err := remote.NewConfig(client)
			if err != nil {
				return err
			}

			cmdCLI := newCliApp(cfg, client)

			cmd.RegisterSignalCallback(utils.SoftKiller(cmdCLI.Kill))

			return describeQuit(cmdCLI.Run(ctx))
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

func newCliApp(rc *remote.Config, client *tequilapi_client.Client) *cliApp {
	dataDir := rc.GetStringByFlag(config.FlagDataDir)
	return &cliApp{
		config:      rc,
		tequilapi:   client,
		historyFile: filepath.Join(dataDir, ".cli_history"),
	}
}

// cliApp describes CLI based Mysterium UI
type cliApp struct {
	config           *remote.Config
	historyFile      string
	tequilapi        *tequilapi_client.Client
	fetchedProposals []contract.ProposalDTO
	completer        *readline.PrefixCompleter
	reader           *readline.Instance

	currentConsumerID string
}

const (
	redColor                  = "\033[31m%s\033[0m"
	identityDefaultPassphrase = ""
	statusConnected           = string(connectionstate.Connected)
	statusNotConnected        = string(connectionstate.NotConnected)
)

var errTermsNotAgreed = errors.New("you must agree with provider and consumer terms of use in order to use this command")

var versionSummary = metadata.VersionAsSummary(metadata.LicenseCopyright(
	"type 'license --warranty'",
	"type 'license --conditions'",
))

func (c *cliApp) handleTOS(ctx *cli.Context) error {
	if ctx.Bool(config.FlagAgreedTermsConditions.Name) {
		c.acceptTOS()
		return nil
	}

	agreedC := c.config.GetBool(contract.TermsConsumerAgreed)

	if !agreedC {
		return errTermsNotAgreed
	}

	agreedP := c.config.GetBool(contract.TermsProviderAgreed)
	if !agreedP {
		return errTermsNotAgreed
	}

	version := c.config.GetString(contract.TermsVersion)
	if version != terms.TermsVersion {
		return fmt.Errorf("you've agreed to terms of use version %s, but version %s is required", version, terms.TermsVersion)
	}

	return nil
}

func (c *cliApp) acceptTOS() {
	t := true
	if err := c.tequilapi.UpdateTerms(contract.TermsRequest{
		AgreedConsumer: &t,
		AgreedProvider: &t,
		AgreedVersion:  terms.TermsVersion,
	}); err != nil {
		clio.Info("Failed to save terms of use agreement, you will have to re-agree on next launch")
	}
}

// Run runs CLI interface synchronously, in the same thread while blocking it
func (c *cliApp) Run(ctx *cli.Context) (err error) {
	if err := c.handleTOS(ctx); err != nil {
		clio.PrintTOSError(err)
		return nil
	}

	c.completer = newAutocompleter(c.tequilapi, c.fetchedProposals)
	c.fetchedProposals = c.fetchProposals()

	if ctx.Args().Len() > 0 {
		return c.handleActions(ctx.Args().Slice())
	}

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

		args, err := shlex.Split(line, true)
		if err != nil {
			return err
		}
		c.handleActions(args)
	}
}

// Kill stops cli
func (c *cliApp) Kill() error {
	c.reader.Clean()
	return c.reader.Close()
}

func (c *cliApp) handleActions(args []string) error {
	if len(args) == 0 {
		return c.help()
	}
	cmd := strings.TrimSpace(args[0])

	cmdArgs := make([]string, 0)
	if len(args) > 1 {
		cmdArgs = args[1:]
	}

	staticCmds := []struct {
		command string
		handler func() error
	}{
		{"exit", c.quit},
		{"quit", c.quit},
		{"help", c.help},
		{"status", c.status},
		{"healthcheck", c.healthcheck},
		{"nat", c.nodeMonitoringStatus},
		{"location", c.location},
		{"disconnect", c.disconnect},
		{"stop", c.stopClient},
		{"version", c.version},
	}

	argCmds := []struct {
		command string
		handler func(args []string) error
	}{
		{"connect", c.connect},
		{"identities", c.identities},
		{"orders", c.order},
		{"license", c.license},
		{"proposals", c.proposals},
		{"service", c.service},
		{"stake", c.stake},
		{"mmn", c.mmnApiKey},
	}

	for _, c := range staticCmds {
		if cmd == c.command {
			err := c.handler()
			if err != nil {
				clio.Error(formatForHuman(err))
			}
			return err
		}
	}

	for _, c := range argCmds {
		if cmd == c.command {
			err := c.handler(cmdArgs)
			if err != nil {
				clio.Error(formatForHuman(err))
			}
			return err
		}
	}

	// Command matched nothing
	return c.help()
}

func (c *cliApp) connect(args []string) (err error) {
	helpMsg := "Please type in the provider identity. connect <consumer-identity> <provider-identity> <service-type> [dns=auto|provider|system|1.1.1.1] [disable-kill-switch]"
	if len(args) < 3 {
		clio.Info(helpMsg)
		return errWrongArgumentCount
	}

	consumerID, providerID, serviceType := args[0], args[1], args[2]
	migrationStatus, err := c.tequilapi.MigrateHermesStatus(consumerID)
	if migrationStatus.Status == contract.MigrationStatusRequired {
		clio.Infof("Hermes migration status: %s\n", migrationStatus.Status)
		clio.Info("Migration started")
		err := c.tequilapi.MigrateHermes(consumerID)
		if err != nil {
			return err
		}
		clio.Info("Migration finished successfully")
		clio.Info("Try to reconnect")
		return nil
	}

	if !services.IsTypeValid(serviceType) {
		return fmt.Errorf("invalid service type, expected one of: %s", strings.Join(services.Types(), ","))
	}

	var disableKillSwitch bool
	var dns connection.DNSOption

	for _, arg := range args[3:] {
		if strings.HasPrefix(arg, "dns=") {
			kv := strings.Split(arg, "=")
			dns, err = connection.NewDNSOption(kv[1])
			if err != nil {
				clio.Info(helpMsg)
				return fmt.Errorf("invalid value: %w", err)
			}
			continue
		}
		switch arg {
		case "disable-kill-switch":
			disableKillSwitch = true
		default:
			clio.Info(helpMsg)
			return errUnknownArgument
		}
	}

	connectOptions := contract.ConnectOptions{
		DNS:               dns,
		DisableKillSwitch: disableKillSwitch,
	}

	clio.Status("CONNECTING", "from:", consumerID, "to:", providerID)

	hermesID, err := c.config.GetHermesID()
	if err != nil {
		return err
	}

	// Dont throw an error here incase user identity has a password on it
	// or we failed to randomly unlock it. We can still try to connect
	// if identity it locked, it will notify us anyway.
	_ = c.tequilapi.Unlock(consumerID, "")

	_, err = c.tequilapi.ConnectionCreate(consumerID, providerID, hermesID, serviceType, connectOptions)
	if err != nil {
		return err
	}

	c.currentConsumerID = consumerID

	clio.Success("Connected.")
	return nil
}

func (c *cliApp) mmnApiKey(args []string) (err error) {
	profileUrl := strings.TrimSuffix(c.config.GetStringByFlag(config.FlagMMNAddress), "/") + "/me"
	usage := "Set MMN's API key and claim this node:\nmmn <api-key>\nTo get the token, visit: " + profileUrl + "\n"

	if len(args) == 0 {
		clio.Info(usage)
		return
	}

	apiKey := args[0]

	err = c.tequilapi.SetMMNApiKey(contract.MMNApiKeyRequest{
		ApiKey: apiKey,
	})
	if err != nil {
		return fmt.Errorf("failed to set MMN API key: %w", err)
	}

	clio.Success("MMN API key configured.")
	return nil
}

func (c *cliApp) disconnect() (err error) {
	err = c.tequilapi.ConnectionDestroy(0)
	if err != nil {
		return err
	}
	c.currentConsumerID = ""
	clio.Success("Disconnected.")
	return nil
}

func (c *cliApp) status() (err error) {
	status, err := c.tequilapi.ConnectionStatus(0)
	if err != nil {
		clio.Warn(err)
	} else {
		clio.Info("Status:", status.Status)
		clio.Info("SID:", status.SessionID)
	}

	ip, err := c.tequilapi.ConnectionIP()
	if err != nil {
		clio.Warn(err)
	} else {
		clio.Info("IP:", ip.IP)
	}

	location, err := c.tequilapi.ConnectionLocation()
	if err != nil {
		clio.Warn(err)
	} else {
		clio.Info(fmt.Sprintf("Location: %s, %s (%s - %s)", location.City, location.Country, location.IPType, location.ISP))
	}

	if status.Status == statusConnected {
		clio.Info("Proposal:", status.Proposal)

		statistics, err := c.tequilapi.ConnectionStatistics()
		if err != nil {
			clio.Warn(err)
		} else {
			clio.Info(fmt.Sprintf("Connection duration: %s", time.Duration(statistics.Duration)*time.Second))
			clio.Info(fmt.Sprintf("Data: %s/%s", datasize.FromBytes(statistics.BytesReceived), datasize.FromBytes(statistics.BytesSent)))
			clio.Info(fmt.Sprintf("Throughput: %s/%s", datasize.BitSpeed(statistics.ThroughputReceived), datasize.BitSpeed(statistics.ThroughputSent)))
			clio.Info(fmt.Sprintf("Spent: %s", money.New(statistics.TokensSpent)))
		}
	}
	return nil
}

func (c *cliApp) healthcheck() (err error) {
	healthcheck, err := c.tequilapi.Healthcheck()
	if err != nil {
		return err
	}

	clio.Info(fmt.Sprintf("Uptime: %v", healthcheck.Uptime))
	clio.Info(fmt.Sprintf("Process: %v", healthcheck.Process))
	clio.Info(fmt.Sprintf("Version: %v", healthcheck.Version))
	buildString := metadata.FormatString(healthcheck.BuildInfo.Commit, healthcheck.BuildInfo.Branch, healthcheck.BuildInfo.BuildNumber)
	clio.Info(buildString)
	return nil
}

func (c *cliApp) nodeMonitoringStatus() (err error) {
	status, err := c.tequilapi.NATStatus()
	if err != nil {
		return fmt.Errorf("failed to retrieve NAT traversal status: %w", err)
	}

	clio.Infof("Node Monitoring Status: %q\n", status.Status)

	connStatus, err := c.tequilapi.ConnectionStatus(0)
	if err != nil {
		clio.Warn(err)
		return
	}

	if connStatus.Status != statusNotConnected {
		return nil
	}
	natType, err := c.tequilapi.NATType()
	switch {
	case err != nil:
		clio.Warn(err)
	case natType.Error != "":
		clio.Warn(natType.Error)
	default:
		displayedNATType, ok := nattype.HumanReadableTypes[natType.Type]
		if !ok {
			displayedNATType = string(natType.Type)
		}
		clio.Info("NAT type:", displayedNATType)
	}

	return nil
}

func (c *cliApp) proposals(args []string) (err error) {
	proposals := c.fetchProposals()
	c.fetchedProposals = proposals

	filter := ""
	if len(args) > 0 {
		filter = strings.Join(args, " ")
	}
	filterMsg := ""
	if filter != "" {
		filterMsg = fmt.Sprintf("(filter: '%s')", filter)
	}
	clio.Info(fmt.Sprintf("Found %v proposals %s", len(proposals), filterMsg))

	for _, proposal := range proposals {
		country := proposal.Location.Country
		if country == "" {
			country = "Unknown"
		}

		var policies []string
		if proposal.AccessPolicies != nil {
			for _, policy := range *proposal.AccessPolicies {
				policies = append(policies, policy.ID)
			}
		}

		msg := fmt.Sprintf("- provider id: %v\ttype: %v\tcountry: %v\taccess policies: %v\tprovider type: %v", proposal.ProviderID, proposal.ServiceType, country, strings.Join(policies, ","), proposal.Location.IPType)

		if filter == "" ||
			strings.Contains(proposal.ProviderID, filter) ||
			strings.Contains(country, filter) {
			clio.Info(msg)
		}
	}

	return nil
}

func (c *cliApp) fetchProposals() []contract.ProposalDTO {
	proposals, err := c.tequilapi.ProposalsNATCompatible()
	if err != nil {
		clio.Warn(err)
		return []contract.ProposalDTO{}
	}
	return proposals
}

func (c *cliApp) location() (err error) {
	location, err := c.tequilapi.OriginLocation()
	if err != nil {
		return err
	}

	clio.Info(fmt.Sprintf("Location: %s, %s (%s - %s)", location.City, location.Country, location.IPType, location.ISP))
	return nil
}

func (c *cliApp) help() (err error) {
	clio.Info("Mysterium CLI commands:")
	fmt.Println(c.completer.Tree("  "))
	return nil
}

// quit stops cli and client commands and exits application
func (c *cliApp) quit() (err error) {
	stop := utils.SoftKiller(c.Kill)
	stop()
	return nil
}

func (c *cliApp) stopClient() (err error) {
	err = c.tequilapi.Stop()
	if err != nil {
		return fmt.Errorf("cannot stop the client: %w", err)
	}
	clio.Success("Client stopped")
	return nil
}

func (c *cliApp) version() (err error) {
	fmt.Println(versionSummary)
	return nil
}

func (c *cliApp) license(args []string) (err error) {
	arg := ""
	if len(args) > 0 {
		arg = args[0]
	}
	if arg == "warranty" {
		fmt.Print(metadata.LicenseWarranty)
	} else if arg == "conditions" {
		fmt.Print(metadata.LicenseConditions)
	} else {
		clio.Info("identities command:\n    warranty\n    conditions")
	}
	return nil
}

func getIdentityOptionList(tequilapi *tequilapi_client.Client) func(string) []string {
	return func(line string) []string {
		var identities []string
		ids, err := tequilapi.GetIdentities()
		if err != nil {
			clio.Warn(err)
			return identities
		}
		for _, id := range ids {
			identities = append(identities, id.Address)
		}

		return identities
	}
}

func getProposalOptionList(proposals []contract.ProposalDTO) func(string) []string {
	return func(line string) []string {
		var providerIDS []string
		for _, proposal := range proposals {
			providerIDS = append(providerIDS, proposal.ProviderID)
		}
		return providerIDS
	}
}

func newAutocompleter(tequilapi *tequilapi_client.Client, proposals []contract.ProposalDTO) *readline.PrefixCompleter {
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
			readline.PcItem("get", readline.PcItemDynamic(getIdentityOptionList(tequilapi))),
			readline.PcItem("balance", readline.PcItemDynamic(getIdentityOptionList(tequilapi))),
			readline.PcItem("new"),
			readline.PcItem("unlock", readline.PcItemDynamic(getIdentityOptionList(tequilapi))),
			readline.PcItem("register", readline.PcItemDynamic(getIdentityOptionList(tequilapi))),
			readline.PcItem("beneficiary-status", readline.PcItemDynamic(getIdentityOptionList(tequilapi))),
			readline.PcItem("beneficiary-set", readline.PcItemDynamic(getIdentityOptionList(tequilapi))),
			readline.PcItem("settle", readline.PcItemDynamic(getIdentityOptionList(tequilapi))),
			readline.PcItem("referralcode", readline.PcItemDynamic(getIdentityOptionList(tequilapi))),
			readline.PcItem("export", readline.PcItemDynamic(getIdentityOptionList(tequilapi))),
			readline.PcItem("import"),
			readline.PcItem("withdraw", readline.PcItemDynamic(getIdentityOptionList(tequilapi))),
			readline.PcItem("last-withdrawal", readline.PcItemDynamic(getIdentityOptionList(tequilapi))),
			readline.PcItem("migrate-hermes", readline.PcItemDynamic(getIdentityOptionList(tequilapi))),
			readline.PcItem("migrate-hermes-status", readline.PcItemDynamic(getIdentityOptionList(tequilapi))),
		),
		readline.PcItem("status"),
		readline.PcItem(
			"stake",
			readline.PcItem("increase"),
			readline.PcItem("decrease"),
		),
		readline.PcItem("orders",
			readline.PcItem("create", readline.PcItemDynamic(getIdentityOptionList(tequilapi))),
			readline.PcItem("get", readline.PcItemDynamic(getIdentityOptionList(tequilapi))),
			readline.PcItem("get-all", readline.PcItemDynamic(getIdentityOptionList(tequilapi))),
			readline.PcItem("gateways"),
		),
		readline.PcItem("healthcheck"),
		readline.PcItem("nat"),
		readline.PcItem("proposals"),
		readline.PcItem("location"),
		readline.PcItem("disconnect"),
		readline.PcItem("mmn"),
		readline.PcItem("help"),
		readline.PcItem("quit"),
		readline.PcItem("stop"),
		readline.PcItem(
			"license",
			readline.PcItem("warranty"),
			readline.PcItem("conditions"),
		),
	)
}

func parseStartFlags(serviceType string, args ...string) (services.StartOptions, error) {
	var flags []cli.Flag
	config.RegisterFlagsServiceStart(&flags)
	config.RegisterFlagsServiceOpenvpn(&flags)
	config.RegisterFlagsServiceWireguard(&flags)
	config.RegisterFlagsServiceNoop(&flags)

	set := flag.NewFlagSet("", flag.ContinueOnError)
	for _, f := range flags {
		f.Apply(set)
	}
	if err := set.Parse(args); err != nil {
		return services.StartOptions{}, err
	}

	ctx := cli.NewContext(nil, set, nil)
	config.ParseFlagsServiceStart(ctx)
	config.ParseFlagsServiceOpenvpn(ctx)
	config.ParseFlagsServiceWireguard(ctx)
	config.ParseFlagsServiceNoop(ctx)

	return services.GetStartOptions(serviceType)
}
