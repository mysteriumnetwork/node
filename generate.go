//go:generate protoc -I=. --go_out=./pb ./pb/ping.proto
//go:generate protoc -I=. --go_out=./pb ./pb/p2p.proto
//go:generate protoc -I=. --go_out=./pb ./pb/session.proto
//go:generate protoc -I=. --go_out=./pb ./pb/payment.proto

package main
