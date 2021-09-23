package contract

import (
	"github.com/mysteriumnetwork/node/core/node"
)

// NodeStatusResponse a node status reflects monitoring agent POV on node availability
// swagger:model NodeStatusResponse
type NodeStatusResponse struct {
	Status node.MonitoringStatus `json:"status"`
}
