package cli

import (
	"fmt"
	"github.com/chzyer/readline"
	tequilapi_client "github.com/mysterium/node/tequilapi/client"
	"io"
	"log"
	"strings"
)

// NewCommand constructs CLI based with possibility to control quiting
func NewCommand(
	historyFile string,
	tequilapi *tequilapi_client.Client,
	quitHandler func() error,
) *Command {
	return &Command{
		historyFile: historyFile,
		tequilapi:   tequilapi,
		completer:   newAutocompleter(tequilapi),
		quitHandler: quitHandler,
	}
}

// Command describes CLI based Mysterium UI
type Command struct {
	historyFile string
	tequilapi   *tequilapi_client.Client
	quitHandler func() error
	completer   *readline.PrefixCompleter
	reader      *readline.Instance
}

const redColor = "\033[31m%s\033[0m"
const identityDefaultPassphrase = ""

// Run starts CLI interface
func (c *Command) Run() (err error) {
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

	return nil
}

//Kill stops tequilapi service
func (c *Command) Kill() error {
	c.reader.Clean()
	err := c.reader.Close()
	if err != nil {
		return err
	}

	return c.quitHandler()
}

func (c *Command) handleActions(line string) {
	line = strings.TrimSpace(line)
	switch {
	case strings.HasPrefix(line, "connect"):
		c.connect(line)
	case line == "exit" || line == "quit":
		c.quit()

	case strings.HasPrefix(line, "unlock"):
		c.unlock(line)

	case line == "help":
		c.help()

	case line == "status":
		c.status()

	case line == "disconnect":
		c.disconnect()

	case strings.HasPrefix(line, "identities"):
		c.identities(line)

	default:
		if len(line) > 0 {
			c.help()
		}
	}
}

func (c *Command) connect(line string) {
	connectionArgs := strings.TrimSpace(line[7:])
	if len(connectionArgs) == 0 {
		info("Press tab to select identity or create a new one. Connect <your-identity> <node-identity>")
		return
	}

	identities := strings.Fields(connectionArgs)

	if len(identities) != 2 {
		info("Please type in the node identity. Connect <your-identity> <node-identity>")
		return
	}

	consumerId, providerId := identities[0], identities[1]

	if consumerId == "new" {
		id, err := c.tequilapi.NewIdentity(identityDefaultPassphrase)
		if err != nil {
			warn(err)
			return
		}
		consumerId = id.Address
		success("New identity created:", consumerId)
	}

	status("CONNECTING", "from:", consumerId, "to:", providerId)

	_, err := c.tequilapi.Connect(consumerId, providerId)
	if err != nil {
		warn(err)
		return
	}

	success("Connected.")
}

func (c *Command) unlock(line string) {
	unlockArgs := strings.TrimSpace(line[7:])

	unlockSignature := "Unlock <identity> [passphrase]"
	if len(unlockArgs) == 0 {
		info("Press tab to select identity.", unlockSignature)
		return
	}

	args := strings.Fields(unlockArgs)
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
		return
	}

	info("Status:", status.Status)
	info("SID:", status.SessionId)
}

func (c *Command) help() {
	info("Mysterium CLI tequilapi commands:")
	fmt.Println(c.completer.Tree("  "))
}

func (c *Command) quit() {
	err := c.Kill()
	if err != nil {
		warn(err)
		return
	}
}

func (c *Command) identities(line string) {
	argsString := strings.TrimSpace(line[10:])
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

func newAutocompleter(tequilapi *tequilapi_client.Client) *readline.PrefixCompleter {
	return readline.NewPrefixCompleter(
		readline.PcItem(
			"connect",
			readline.PcItemDynamic(
				getIdentityOptionList(tequilapi),
			),
		),
		readline.PcItem(
			"identities",
			readline.PcItem("new"),
			readline.PcItem("list"),
		),
		readline.PcItem("status"),
		readline.PcItem("disconnect"),
		readline.PcItem("help"),
		readline.PcItem("quit"),
		readline.PcItem(
			"unlock",
			readline.PcItemDynamic(
				getIdentityOptionList(tequilapi),
			),
		),
	)
}
