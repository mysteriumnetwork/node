package dto

type Contact struct {
	Type       string            `json:"type"`
	Definition ContactDefinition `json:"definition"`
}

type ContactDefinition interface{}




