package cli

import (
	"math/big"
	"strings"

	"github.com/urfave/cli/v2"

	"github.com/spf13/cast"

	"github.com/mysteriumnetwork/node/tequilapi/client"

	"github.com/mysteriumnetwork/node/config"
	"github.com/rs/zerolog/log"
)

const (
	defaultTequilApiAddress = "localhost"
	defaultTequilApiPort    = 4050
)

var rConfig = newRemoteConfig()

type remoteConfig struct {
	config map[string]interface{}
}

func newRemoteConfig() *remoteConfig {
	return &remoteConfig{}
}

func refreshRemoteConfig(client *client.Client) error {
	config, err := client.FetchConfig()
	if err != nil {
		return err
	}
	rConfig.config = config
	return nil
}

// Get returns stored remote config value as-is.
func (rc *remoteConfig) Get(key string) interface{} {
	segments := strings.Split(strings.ToLower(key), ".")
	value := config.SearchMap(rc.config, segments)
	log.Debug().Msgf("Returning remote config value %v:%v", key, value)
	return value
}

// GetStringByFlag shorthand for getting current configuration value for cli.BoolFlag.
func (rc *remoteConfig) GetStringByFlag(flag cli.StringFlag) string {
	return rc.GetString(flag.Name)
}

// GetString returns config value as string.
func (rc *remoteConfig) GetString(key string) string {
	return cast.ToString(rc.Get(key))
}

// GetBoolByFlag shorthand for getting current configuration value for cli.BoolFlag.
func (rc *remoteConfig) GetBoolByFlag(flag cli.BoolFlag) bool {
	return rc.GetBool(flag.Name)
}

// GetBool returns config value as bool.
func (rc *remoteConfig) GetBool(key string) bool {
	return cast.ToBool(rc.Get(key))
}

// GetBigIntByFlag shorthand for getting and parsing a configuration value for cli.StringFlag that's a big.Int.
func (rc *remoteConfig) GetBigIntByFlag(flag cli.StringFlag) *big.Int {
	return rc.GetBigInt(flag.Name)
}

// GetBigInt returns config value as big.Int.
func (rc *remoteConfig) GetBigInt(key string) *big.Int {
	b, _ := new(big.Int).SetString(rc.GetString(key), 10)
	return b
}
