package dto

import dto_discovery "github.com/mysterium/node/service_discovery/dto"

type NodeRegisterRequest struct {
	ServiceProposal dto_discovery.ServiceProposal `json:"service_proposal"`
}
