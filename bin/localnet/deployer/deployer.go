package main

import (
	"flag"
	"fmt"
	"github.com/MysteriumNetwork/payments/cli/helpers"
	"github.com/MysteriumNetwork/payments/mysttoken/generated"
	generated2 "github.com/MysteriumNetwork/payments/promises/generated"
	"math/big"
	"os"
)

func main() {
	flag.Parse()

	ks := helpers.GetKeystore()

	acc, err := helpers.GetUnlockedAcc(ks)
	checkError(err)

	transactor := helpers.CreateNewKeystoreTransactor(ks, acc)

	client, synced, err := helpers.LookupBackend()
	checkError(err)
	<-synced

	mystTokenAddress, tx, _, err := generated.DeployMystToken(transactor, client)
	checkError(err)
	fmt.Println("Token: ", mystTokenAddress.String())

	transactor.Nonce = big.NewInt(int64(tx.Nonce() + 1))
	paymentsAddress, _, _, err := generated2.DeployIdentityPromises(transactor, client, mystTokenAddress, big.NewInt(100))
	checkError(err)

	fmt.Println("Payments: ", paymentsAddress.String())
}

func checkError(err error) {
	if err != nil {
		fmt.Println("Error: ", err.Error())
		os.Exit(1)
	}
}
