package dto

import "github.com/mysterium/node/service_discovery/dto"

type NodeRegisterRequest struct {
	ServiceProposal dto.ServiceProposal `json:"service_proposal"`
}
