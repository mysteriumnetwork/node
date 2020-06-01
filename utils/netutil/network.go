/*
 * Copyright (C) 2020 The "MysteriumNetwork/node" Authors.
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

package netutil

import (
	"net"
	"strings"

	"github.com/jackpal/gateway"
	"github.com/mysteriumnetwork/node/core/storage/boltdb"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var defaultRouteManager *routeManager = nil

const (
	routeRecordDelimeter = "|"
	routeRecordBucket    = "exclude_route"
)

type route struct {
	Record string `storm:"id"`
}

type routeManager struct {
	db *boltdb.Bolt
}

// SetRouteManagerStorage initiate defaultRouteManager with a provided storage.
func SetRouteManagerStorage(db *boltdb.Bolt) {
	defaultRouteManager = &routeManager{
		db: db,
	}
}

// ClearStaleRoutes removes stale route from the host routing table.
func ClearStaleRoutes() {
	if defaultRouteManager == nil {
		return
	}

	var records []route
	err := defaultRouteManager.db.GetAllFrom(routeRecordBucket, &records)

	if err != nil {
		log.Error().Err(err).Msgf("Failed to get %s records", routeRecordBucket)
		return
	}

	for _, r := range records {
		args := strings.Split(r.Record, routeRecordDelimeter)

		if len(args) != 2 {
			log.Error().Err(err).Msgf("Failed to parse %s record", r.Record)
		} else {
			log.Info().Msgf("Cleaning stale route: %s %s", args[0], args[1])
			if err := deleteRoute(args[0], args[1]); err != nil {
				log.Error().Err(err).Msgf("Failed to delete route: %s %s", args[0], args[1])
			}
		}

		err := defaultRouteManager.db.Delete(routeRecordBucket, &r)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to delete %s record", r.Record)
		}
	}
}

// ExcludeRoute excludes given IP from VPN tunnel.
func ExcludeRoute(ip net.IP) error {
	gw, err := gateway.DiscoverGateway()
	if err != nil {
		return err
	}

	if defaultRouteManager != nil {
		err := defaultRouteManager.db.Store(routeRecordBucket, &route{
			Record: strings.Join([]string{ip.String(), gw.String()}, routeRecordDelimeter),
		})
		if err != nil {
			log.Error().Err(err).Msgf("Failed to save %s record", routeRecordBucket)
		}
	}

	return excludeRoute(ip, gw)
}

// AddDefaultRoute adds default VPN tunnel route.
func AddDefaultRoute(iface string) error {
	return addDefaultRoute(iface)
}

// AssignIP assigns subnet to given interface.
func AssignIP(iface string, subnet net.IPNet) error {
	return assignIP(iface, subnet)
}

// LogNetworkStats logs network information to the Trace log level.
func LogNetworkStats() {
	if log.Logger.GetLevel() != zerolog.TraceLevel {
		return
	}

	logNetworkStats()
}

func logOutputToTrace(out []byte, err error, args ...string) {
	logSkipFrame := log.With().CallerWithSkipFrameCount(3).Logger()

	if err != nil {
		(&logSkipFrame).Trace().Msgf("Failed to get %s error: %v", strings.Join(args, " "), err)
	} else {
		(&logSkipFrame).Trace().Msgf("%q output:\n%s", strings.Join(args, " "), out)
	}
}
