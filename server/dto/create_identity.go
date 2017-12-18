package dto

import dto_discovery "github.com/mysterium/node/service_discovery/dto"

type CreateIdentityRequest struct {
	Identity dto_discovery.Identity `json:"identity"`
}
