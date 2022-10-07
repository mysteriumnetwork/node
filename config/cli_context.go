package config

import "time"

type CliContext interface {
	IsSet(name string) bool
	String(name string) string

	StringSlice(name string) []string
	Bool(name string) bool
	Int(name string) int
	Uint64(name string) uint64
	Int64(name string) int64
	Float64(name string) float64
	Duration(name string) time.Duration
}

