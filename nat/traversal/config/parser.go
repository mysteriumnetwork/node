package config

import (
	"encoding/json"

	"github.com/mysteriumnetwork/node/services/openvpn"
	"github.com/pkg/errors"
)

type ConsumerConfigParser struct {
}

func NewConfigParser() *ConsumerConfigParser {
	return &ConsumerConfigParser{}
}

func (c *ConsumerConfigParser) Parse(config json.RawMessage) (ip string, port int, err error) {
	// TODO: since we are getting json.RawMessage here and not interface{} type not sure how to handle multiple services
	// since NATPinger is one for all services and we get config from communication channel where service type is not know yet.
	var cfg openvpn.ConsumerConfig
	err = json.Unmarshal(config, &cfg)
	if err != nil {
		return "", 0, errors.Wrap(err, "parsing consumer address:port failed")
	}
	return cfg.IP, cfg.Port, nil
}
