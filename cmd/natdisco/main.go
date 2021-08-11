/*
 * Copyright (C) 2021 The "MysteriumNetwork/node" Authors.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/mysteriumnetwork/node/nat"

	"github.com/mysteriumnetwork/node/nat/behavior"
)

const addressListSeparator = ","

var (
	addressList  = flag.String("servers", "stun.mysterium.network:3478,stun.stunprotocol.org:3478,stun.sip.us:3478", "comma-separated list of STUN servers")
	reqTimeout   = flag.Duration("req-timeout", 1*time.Second, "timeout to wait for each STUN server response")
	totalTimeout = flag.Duration("total-timeout", 10*time.Second, "overall operation deadline")
	raw          = flag.Bool("raw", false, "print raw NAT_TYPE_* value")
)

func run() int {
	flag.Parse()

	var addresses []string
	for _, address := range strings.Split(*addressList, addressListSeparator) {
		address = strings.TrimSpace(address)
		if address != "" {
			addresses = append(addresses, address)
		}
	}

	ctx, cl := context.WithTimeout(context.Background(), *totalTimeout)
	defer cl()
	res, err := behavior.RacingDiscoverNATBehavior(ctx, addresses, *reqTimeout)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}
	if *raw {
		fmt.Println(res)
	} else {
		fmt.Println("NAT Type:", nat.HumanReadableTypes[res])
	}
	return 0
}

func main() {
	os.Exit(run())
}
