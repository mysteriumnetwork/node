package contract

// MystnodesSSOLinkResponse contains a link to initiate auth via mystnodes
// swagger:model MystnodesSSOLinkResponse
type MystnodesSSOLinkResponse struct {
	Link string `json:"link"`
}

// MystnodesSSOGrantVerificationRequest
type MystnodesSSOGrantVerificationRequest struct {
	AuthorizationGrant    string `json:"authorizationGrant"`
	CodeVerifierBase64url string `json:"codeVerifierBase64url"`
}

// MystnodesSSOGrantLoginRequest
type MystnodesSSOGrantLoginRequest struct {
	AuthorizationGrant string `json:"authorization_grant"`
}
