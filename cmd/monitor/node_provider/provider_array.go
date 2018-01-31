package node_provider

func NewArrayProvider(nodeKeys []string) NodeProvider {
	return &arrayProvider{
		nodeKeys: nodeKeys,
	}
}

type arrayProvider struct {
	nodeKeys []string
}

func (provider *arrayProvider) WithEachNode(consumer NodeConsumer) {
	for _, nodeKey := range provider.nodeKeys {
		consumer(nodeKey)
	}
}

func (provider *arrayProvider) Close() error {
	return nil
}
