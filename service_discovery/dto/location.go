package dto

type Location struct {
	Country string `json:"country,omitempty"`
	City    string `json:"city,omitempty"`
	// Autonomous System Number http://www.whatismyip.cx/
	ASN string `json:"asn,omitempty"`
}
