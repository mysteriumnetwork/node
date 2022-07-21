package data_transfer

import "github.com/mysteriumnetwork/node/market"

// Bootstrap is called on program initialization time and registers various deserializers related to data_transfer service
func Bootstrap() {
	market.RegisterServiceType(ServiceType)
}
