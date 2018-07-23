package blockchain

import (
	"github.com/ethereum/go-ethereum/ethclient"
)

func NewClient(rpcUrl string) (*ethclient.Client, error) {
	return ethclient.Dial(rpcUrl)
}
