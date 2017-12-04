package session

type SessionCreateRequest struct {
	ProposalId int `json:"proposal_id"`
}

type SessionCreateResponse struct {
	Success bool       `json:"success"`
	Message string     `json:"message"`
	Session VpnSession `json:"session"`
}
