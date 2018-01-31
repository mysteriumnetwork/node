package session

import "github.com/satori/go.uuid"

// UUIDGenerator generates session ids based on random UUIDs
type UUIDGenerator struct{}

// Generate method returns SessionID based on random UUID
func (generator *UUIDGenerator) Generate() SessionID {
	return SessionID(uuid.NewV4().String())
}
