package node_provider

type NodeProvider interface {
	WithEachNode(consumer NodeConsumer)
	Close() error
}

type NodeConsumer func(nodeKey string)
