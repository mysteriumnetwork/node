package dto

type Location struct {
	Country string `json:"country"`
	City    string `json:"city"`
	// Autonomous System Number http://www.whatismyip.cx/
	ASN string `json:"asn"`
}
