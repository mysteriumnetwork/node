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

	"github.com/mysteriumnetwork/node/core/port"
)

const addressListSeparator = ","

var (
	serverList   = flag.String("server-list", "vm-0.com:4589", "comma-separated list of asymmetric UDP echo servers")
	reqTimeout   = flag.Duration("req-timeout", 2*time.Second, "timeout to wait for UDP response")
	checkedPort  = flag.Int("port", 12345, "checked port")
	totalTimeout = flag.Duration("total-timeout", 5*time.Second, "overall operation deadline")
)

func run() int {
	flag.Parse()

	var addresses []string
	for _, address := range strings.Split(*serverList, addressListSeparator) {
		address = strings.TrimSpace(address)
		if address != "" {
			addresses = append(addresses, address)
		}
	}

	ctx, cl := context.WithTimeout(context.Background(), *totalTimeout)
	defer cl()
	res, err := port.GloballyReachable(ctx, port.Port(*checkedPort), addresses, *reqTimeout)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}
	fmt.Println(res)
	return 0
}

func main() {
	os.Exit(run())
}
