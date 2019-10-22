/*
 * Copyright (C) 2017 The "MysteriumNetwork/node" Authors.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package e2e

import (
	"context"
	"flag"
	"fmt"
	"time"

	"github.com/cheggaaa/pb/v3"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/ethclient"
	tequilapi_client "github.com/mysteriumnetwork/node/tequilapi/client"
)

// Flags to run a test
var (
	ehtRpcUrl = flag.String("geth.url", "http://localhost:8545", "RPC url of ethereum node")
)

// Provider flags
var (
	providerTequilapiHost = flag.String("provider.tequilapi-host", "localhost", "Specify Tequilapi host for provider")
	providerTequilapiPort = flag.Int("provider.tequilapi-port", 4050, "Specify Tequilapi port for provider")
)

// Consumer flags
var (
	consumerTequilapiHost = flag.String("consumer.tequilapi-host", "localhost", "Specify Tequilapi host for consumer")
	consumerTequilapiPort = flag.Int("consumer.tequilapi-port", 4050, "Specify Tequilapi port for consumer")
	consumerServices      = flag.String("consumer.services", "openvpn,noop,wireguard", "Comma separated list of services to try and use")
)

func newTequilapiConsumer() *tequilapi_client.Client {
	return tequilapi_client.NewClient(*consumerTequilapiHost, *consumerTequilapiPort)
}

func newTequilapiProvider() *tequilapi_client.Client {
	return tequilapi_client.NewClient(*providerTequilapiHost, *providerTequilapiPort)
}

func newEthClient() (*ethclient.Client, error) {
	ethClient, synced, err := lookupBackend(*ehtRpcUrl)
	if err != nil {
		return nil, err
	}
	<-synced //wait for sync to finish if any
	return ethClient, nil
}

func lookupBackend(rpcURL string) (*ethclient.Client, chan bool, error) {
	ethClient, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, nil, err
	}

	block, err := ethClient.BlockByNumber(context.Background(), nil)
	if err != nil {
		return nil, nil, err
	}
	fmt.Println("Latest known block is: ", block.NumberU64())

	progress, err := ethClient.SyncProgress(context.Background())
	if err != nil {
		return nil, nil, err
	}
	completed := make(chan bool)
	if progress != nil {
		fmt.Println("Client is in syncing state - any operations will be delayed until finished")
		go trackGethProgress(ethClient, progress, completed)
	} else {
		fmt.Println("Geth process fully synced")
		close(completed)
	}

	return ethClient, completed, nil
}

func trackGethProgress(client *ethclient.Client, lastProgress *ethereum.SyncProgress, completed chan<- bool) {
	bar := pb.New64(int64(lastProgress.HighestBlock)).
		SetTotal(int64(lastProgress.CurrentBlock)).
		Start()
	defer bar.Finish()
	defer close(completed)
	for {
		progress, err := client.SyncProgress(context.Background())
		if err != nil {
			fmt.Println("Error querying client progress: " + err.Error())
			return
		}
		if progress == nil {
			bar.Finish()
			fmt.Println("Client in fully synced state. Proceeding...")
			return
		}
		bar.Set(int(progress.CurrentBlock), true)
		bar.SetTotal(int64(progress.HighestBlock))
		time.Sleep(10 * time.Second)
	}
}
