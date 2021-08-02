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
	"time"

	"github.com/mysteriumnetwork/node/nat/behavior"
)

var (
	address      = flag.String("server", "stun.stunprotocol.org:3478", "STUN server address")
	reqTimeout   = flag.Duration("req-timeout", 3*time.Second, "timeout to wait for each STUN server response")
	totalTimeout = flag.Duration("total-timeout", 15*time.Second, "overall operation deadline")
	raw          = flag.Bool("raw", false, "print raw NAT_TYPE_* value")

	humanReadableTypes = map[string]string{
		behavior.NATTypeNone:               "None",
		behavior.NATTypeFullCone:           "Full Cone",
		behavior.NATTypeRestrictedCone:     "Restricted Cone",
		behavior.NATTypePortRestrictedCone: "Port Restricted Cone",
		behavior.NATTypeSymmetric:          "Symmetric",
	}
)

func run() int {
	flag.Parse()

	ctx, cl := context.WithTimeout(context.Background(), *totalTimeout)
	defer cl()
	res, err := behavior.DiscoverNATBehavior(ctx, *address, *reqTimeout)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}
	if *raw {
		fmt.Println(res)
	} else {
		fmt.Println("NAT Type:", humanReadableTypes[res])
	}
	return 0
}

func main() {
	os.Exit(run())
}
