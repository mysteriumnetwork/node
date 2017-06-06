package node_provider

import (
	"io/ioutil"
	"os"

	log "github.com/cihub/seelog"
)

const MYSTERIUM_PROVIDER_LOG_PREFIX = "[Mysterium.monitor] "

func NewRememberProvider(providerOriginal NodeProvider, statusFilePath string) NodeProvider {
	return &rememberProvider{
		providerOriginal: providerOriginal,
		rememberFile:     statusFilePath,
	}
}

type rememberProvider struct {
	providerOriginal NodeProvider
	rememberFile     string
}

func (provider *rememberProvider) WithEachNode(consumer NodeConsumer) {
	rememberNodeKey := provider.rememberGetNodeKey()

	rememberNodeReached := true
	if rememberNodeKey != "" {
		log.Info(MYSTERIUM_PROVIDER_LOG_PREFIX, "Continuing from remembered node: ", rememberNodeKey)
		rememberNodeReached = false
	}

	provider.providerOriginal.WithEachNode(func(nodeKey string) {
		if rememberNodeReached {
			consumer(nodeKey)
			provider.rememberSaveNodeKey(nodeKey)
		} else {
			rememberNodeReached = rememberNodeKey == nodeKey
		}
	})

	provider.rememberForget()
}

func (provider *rememberProvider) Close() error {
	return provider.providerOriginal.Close()
}

func (provider *rememberProvider) rememberGetNodeKey() string {
	content, err := ioutil.ReadFile(provider.rememberFile)
	if err != nil {
		log.Warn(MYSTERIUM_PROVIDER_LOG_PREFIX, "Cant remember from status file: ", err)
		return ""
	}

	return string(content)
}

func (provider *rememberProvider) rememberSaveNodeKey(nodeKey string) {
	err := ioutil.WriteFile(provider.rememberFile, []byte(nodeKey), 0666)
	if err != nil {
		log.Warn(MYSTERIUM_PROVIDER_LOG_PREFIX, "Cant remember to status file: ", err)
	}
}

func (provider *rememberProvider) rememberForget() {
	os.Remove(provider.rememberFile)
}
