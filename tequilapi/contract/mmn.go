package contract

// MMNApiKeyRequest request used to manage MMN's API key.
// swagger:model MMNApiKeyRequest
type MMNApiKeyRequest struct {
	ApiKey string `json:"api_key,"`
}
