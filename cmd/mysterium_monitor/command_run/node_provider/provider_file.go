package node_provider

import (
	"bufio"
	"os"
)

func NewFileProvider(filePath string) (NodeProvider, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}

	provider := &fileProvider{
		file: file,
	}
	return provider, nil
}

type fileProvider struct {
	file *os.File
}

func (provider *fileProvider) WithEachNode(consumer NodeConsumer) {
	scanner := bufio.NewScanner(provider.file)
	for scanner.Scan() {
		consumer(scanner.Text())
	}
}

func (provider *fileProvider) Close() error {
	return provider.file.Close()
}
