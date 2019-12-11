package mysterium

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type cb struct {
}

func (c cb) OnChange(identityAddress string, balance int64) {
	fmt.Println("Balance change", balance)
}

func TestMobileNode_GetIdentity(t *testing.T) {
	node, err := NewNode("/Users/anjmao/go/src/github.com/mysteriumnetwork/node/build", DefaultLogOptions(), DefaultNetworkOptions())
	assert.NoError(t, err)

	node.RegisterBalanceChangeCallback(&cb{})

	ide, err := node.GetIdentity()
	assert.NoError(t, err)

	fmt.Println("IdentityAddress", ide.IdentityAddress)
	fmt.Println("ChannelAddress", ide.ChannelAddress)
	fmt.Println("Registered", ide.Registered)
	//time.Sleep(3 * time.Second)
	//bal, err := node.GetBalance(&GetBalanceRequest{IdentityAddress: ide.IdentityAddress})
	//assert.NoError(t, err)
	//fmt.Println("Balance", bal.Balance)

	//
	//fees, err := node.GetIdentityRegistrationFees()
	//assert.NoError(t, err)
	//fmt.Println("Fee", fees.Fee)
	//
	//err = node.RegisterIdentity(&RegisterIdentityRequest{
	//	IdentityAddress: ide.IdentityAddress,
	//	Fee:             fees.Fee,
	//})
	//assert.NoError(t, err)
	//
	time.Sleep(5 * time.Second)
}
