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

package nat

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/mysteriumnetwork/node/config"
	"github.com/mysteriumnetwork/node/core/port"
)

const (
	reqTimeout   = 2 * time.Second
	totalTimeout = 5 * time.Second
)

type checkResp struct {
	ok  bool
	err error
}

func checkAllPorts(ports []int) error {
	var addresses []string

	for _, address := range strings.Split(config.GetString(config.FlagPortCheckServers), ",") {
		address = strings.TrimSpace(address)
		if address != "" {
			addresses = append(addresses, address)
		}
	}

	var wg sync.WaitGroup

	wg.Add(len(ports))

	ctx, cancel := context.WithTimeout(context.Background(), totalTimeout)
	defer cancel()

	ch := make(chan checkResp, len(ports))

	for _, p := range ports {
		go func(p int) {
			defer wg.Done()

			ok, err := port.GloballyReachable(ctx, port.Port(p), addresses, reqTimeout)
			ch <- checkResp{
				ok:  ok,
				err: err,
			}
		}(p)
	}

	wg.Wait()
	close(ch)

	for resp := range ch {
		if !resp.ok || resp.err != nil {
			return fmt.Errorf("local port not reachable: %w", resp.err)
		}
	}

	log.Debug().Msgf("All ports %v are reachable", ports)

	return nil
}
