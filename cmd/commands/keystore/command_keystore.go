package keystore

import (
	"fmt"
	"io/ioutil"
	stdlog "log"
	"os"
	"strings"

	"github.com/chzyer/readline"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/mysteriumnetwork/node/cmd/commands/cli/clio"
	"github.com/mysteriumnetwork/node/config"
	"github.com/mysteriumnetwork/node/core/node"
	"github.com/mysteriumnetwork/node/eventbus"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/tequilapi/contract"
	"github.com/mysteriumnetwork/node/utils"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
)

func NewCommand() *cli.Command {
	return &cli.Command{
		Name:  "keystore",
		Usage: "keystore CLI for import / export of private keys",
		Flags: []cli.Flag{},
		Action: func(ctx *cli.Context) error {

			config.ParseFlagsNode(ctx)
			cmdCLI := newCliApp()
			cmdCLI.Run(ctx)

			return nil
		},
	}
}

type cliApp struct {
	fetchedProposals  []contract.ProposalDTO
	completer         *readline.PrefixCompleter
	reader            *readline.Instance
	currentConsumerID string

	ks            *keystore.KeyStore
	bus           eventbus.EventBus
	importer      *identity.Importer
	ex            *identity.Exporter
	signerFactory identity.SignerFactory
}

const (
	redColor = "\033[31m%s\033[0m"
)

func newCliApp() *cliApp {
	log.Logger = log.Logger.Level(zerolog.ErrorLevel)

	opt := node.GetOptions()
	ks := keystore.NewKeyStore(opt.Directories.Keystore, keystore.LightScryptN, keystore.LightScryptP)
	signerFactory := func(id identity.Identity) identity.Signer {
		return identity.NewSigner(ks, id)
	}
	bus := eventbus.New()
	importer := identity.NewImporter(ks, bus, signerFactory)
	exporter := identity.NewExporter(identity.NewKeystoreFilesystem(opt.Directories.Keystore, ks))

	return &cliApp{
		ks:            ks,
		signerFactory: signerFactory,
		bus:           bus,
		importer:      importer,
		ex:            exporter,
	}
}

func (c *cliApp) quit() {
	stop := utils.SoftKiller(c.Kill)
	stop()
}

// Kill stops cli
func (c *cliApp) Kill() error {
	c.reader.Clean()
	return c.reader.Close()
}

func (c *cliApp) Run(ctx *cli.Context) (err error) {
	c.completer = c.newAutocompleter()

	c.reader, err = readline.NewEx(&readline.Config{
		Prompt:          fmt.Sprintf(redColor, "» "),
		AutoComplete:    c.completer,
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
	})
	if err != nil {
		return err
	}

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
	return nil
}

func (c *cliApp) newAutocompleter() *readline.PrefixCompleter {
	return readline.NewPrefixCompleter(
		readline.PcItem("keystore",
			readline.PcItem("list"),
			readline.PcItem("export", readline.PcItemDynamic(c.getIdentityOptionList())),
			readline.PcItem("import"),
		),
		readline.PcItem("quit"),
	)
}

func (c *cliApp) getIdentityOptionList() func(string) []string {
	return func(line string) []string {
		var identities []string

		for _, id := range c.ks.Accounts() {
			identities = append(identities, id.Address.String())
		}

		return identities
	}
}

func (c *cliApp) handleActions(line string) {
	line = strings.TrimSpace(line)

	argCmds := []struct {
		command string
		handler func(argsString string)
	}{
		{"keystore", c.keystoreHandler},
		{"quit", func(_ string) {
			c.quit()
		}},
	}

	for _, cmd := range argCmds {
		if strings.HasPrefix(line, cmd.command) {
			argsString := strings.TrimSpace(line[len(cmd.command):])
			cmd.handler(argsString)
			return
		}
	}
}

func (c *cliApp) keystoreHandler(argsString string) {
	usage := strings.Join([]string{
		"Usage: keystoreHandler <action> [args]",
		"Available actions:",
		"  " + usageListIdentities,
		"  " + usageExportIdentity,
		"  " + usageImportIdentity,
	}, "\n")

	if len(argsString) == 0 {
		clio.Info(usage)
		return
	}

	args := strings.Fields(argsString)
	action := args[0]
	actionArgs := args[1:]

	switch action {
	case "list":
		c.listIdentities(actionArgs)
	case "export":
		c.exportIdentities(actionArgs)
	case "import":
		c.importIdentities(actionArgs)
	default:
		clio.Warnf("Unknown sub-command '%s'\n", argsString)
	}
}

const usageListIdentities = "list"

func (c *cliApp) listIdentities(args []string) {
	if len(args) > 0 {
		clio.Info("Usage: " + usageListIdentities)
		return
	}
	for _, id := range c.ks.Accounts() {
		clio.Status("+", id.Address)
	}
}

const usageExportIdentity = "export <identity> <new_passphrase> [file]"

func (c *cliApp) exportIdentities(actionsArgs []string) {
	if len(actionsArgs) < 2 || len(actionsArgs) > 3 {
		clio.Info("Usage: " + usageExportIdentity)
		return
	}
	id := actionsArgs[0]
	passphrase := actionsArgs[1]
	if len(passphrase) < 12 {
		clio.Error("Passphrase should be 12 symbols at least")
		return
	}

	blob, err := c.ex.Export(id, "", passphrase)
	if err != nil {
		clio.Error("Failed to export identity: ", err)
		return
	}

	if len(actionsArgs) == 3 {
		filepath := actionsArgs[2]
		write := func() error {
			f, err := os.Create(filepath)
			if err != nil {
				return err
			}
			defer f.Close()

			_, err = f.Write(blob)
			return err
		}

		err := write()
		if err != nil {
			clio.Error(fmt.Sprintf("Failed to write exported key to file: %s reason: %s", filepath, err.Error()))
			return
		}

		clio.Success("Identity exported to file:", filepath)
		return
	}

	clio.Success("Private key exported: ")
	fmt.Println(string(blob))
}

const usageImportIdentity = "import <passphrase> <key-string/key-file>"

func (c *cliApp) importIdentities(actionsArgs []string) {
	if len(actionsArgs) != 2 {
		clio.Info("Usage: " + usageImportIdentity)
		return
	}

	key := actionsArgs[1]
	passphrase := actionsArgs[0]

	blob := []byte(key)
	if _, err := os.Stat(key); err == nil {
		blob, err = ioutil.ReadFile(key)
		if err != nil {
			clio.Error(fmt.Sprintf("Can't read provided file: %s reason: %s", key, err.Error()))
			return
		}
	}

	id, err := c.importer.Import(blob, passphrase, "")
	if err != nil {
		clio.Error("Failed to import identity: ", err)
		return
	}
	clio.Success("Identity imported:", id.Address)
}
