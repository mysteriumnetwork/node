//+build !linux

package vnd

import (
	"errors"

	"github.com/mysteriumnetwork/node/firewall"
)

func SetupVendor() (firewall.Vendor, error) {
	return nil, errors.New("this OS doesn't support kill switch")
}
