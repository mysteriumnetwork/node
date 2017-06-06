package node_provider

import (
	"io/ioutil"
	"os"

	log "github.com/cihub/seelog"
)

const MYSTERIUM_PROVIDER_LOG_PREFIX = "[Mysterium.monitor] "

func NewRememberProvider(providerOriginal NodeProvider, statusFilePath string) (NodeProvider, error) {
	rememberFile, err := os.OpenFile(statusFilePath, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return nil, err
	}

	provider := &rememberProvider{
		providerOriginal: providerOriginal,
		rememberFile:     rememberFile,
	}
	return provider, nil
}

type rememberProvider struct {
	providerOriginal NodeProvider
	rememberFile     *os.File
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
	content, err := ioutil.ReadAll(provider.rememberFile)
	if err != nil {
		log.Warn(MYSTERIUM_PROVIDER_LOG_PREFIX, "Cant remember from status file: ", err)
		return ""
	}

	return string(content)
}

func (provider *rememberProvider) rememberSaveNodeKey(nodeKey string) {
	_, err := provider.rememberFile.WriteString(nodeKey)
	if err == nil {
		err = provider.rememberFile.Sync()
	}
	if err != nil {
		log.Warn(MYSTERIUM_PROVIDER_LOG_PREFIX, "Cant remember to status file: ", err)
	}
}

func (provider *rememberProvider) rememberForget() {
	os.Remove(provider.rememberFile.Name())
}
