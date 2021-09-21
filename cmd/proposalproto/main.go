package main

import (
	"encoding/json"
	"fmt"

	"google.golang.org/protobuf/proto"

	"github.com/mysteriumnetwork/node/pb"
)

func main() {
	proposal := pb.Proposal{
		Compatibility: 1,
		ProviderId:    make([]byte, 20),
		ServiceType:   "wireguard",
		Location: &pb.Location{
			Country: "SE",
			IpType:  3,
		},
		Contacts: []string{
			"nats://testnet3-broker.mysterium.network:4222",
		},
		Quality: &pb.Quality{
			Quality:   0.5,
			Latency:   100,
			Bandwidth: 4000000,
		},
	}

	jsonMsg, err := json.MarshalIndent(&proposal, "", "  ")
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("Proposal (json): %s\n", string(jsonMsg))

	msg, err := proto.Marshal(&proposal)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("Proposal length (protobuf):", len(msg))

}
