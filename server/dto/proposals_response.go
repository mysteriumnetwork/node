package dto

import "github.com/mysterium/node/service_discovery/dto"

type ProposalsResponse struct {
	Proposals []dto.ServiceProposal `json:"proposals"`
}
