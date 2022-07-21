package scraping

import "github.com/mysteriumnetwork/node/market"

// Bootstrap is called on program initialization time and registers various deserializers related to scraping service
func Bootstrap() {
	market.RegisterServiceType(ServiceType)
}
