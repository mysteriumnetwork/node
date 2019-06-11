package vnd

import (
	"errors"

	"github.com/mysteriumnetwork/node/firewall"
)

func SetupVendor() (firewall.Vendor, error) {
	return nil, errors.New("android uses its own traffic blocking")
}
