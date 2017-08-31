package dto

type Contact struct {
	Type       string
	Definition ContactDefinition
}

type ContactDefinition interface{}
