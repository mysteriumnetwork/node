package dto

import dto_distovery "github.com/mysterium/node/service_discovery/dto"

type Session struct {
	Id              string                        `json:"session_key"`
	ServiceProposal dto_distovery.ServiceProposal `json:"service_proposal"`
}
