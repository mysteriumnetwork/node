package dto

type Identity string

// shall we change Identity string for Identity struct?
type IdentityData struct {
	Id Identity `json:"id"`
}
type Identities struct {
	Identities []IdentityData `json:"identities"`
}
