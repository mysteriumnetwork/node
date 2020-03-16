//go:generate protoc -I=. --go_out=./pb ./pb/ping.proto
//go:generate protoc -I=. --go_out=./pb ./pb/p2p.proto

package main
