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

package ip

import (
	"errors"
	"io"
	"net"
	"strings"

	"github.com/mysteriumnetwork/node/requests"
	"github.com/mysteriumnetwork/node/utils/random"
)

// IPFallbackAddresses represents the various services we can use to fetch our public IP.
var IPFallbackAddresses = []string{
	"https://api.ipify.org",
	"https://ip2location.io/ip",
	"https://ipinfo.io/ip",
	"https://api.ipify.org",
	"https://ifconfig.me",
	"https://www.trackip.net/ip",
	"https://checkip.amazonaws.com/",
	"https://icanhazip.com",
	"https://ipecho.net/plain",
	"https://ident.me/",
	"http://whatismyip.akamai.com/",
}

var rng = random.NewTimeSeededRand()

func shuffleStringSlice(slice []string) []string {
	tmp := make([]string, len(slice))
	copy(tmp, slice)
	rng.Shuffle(len(tmp), func(i, j int) {
		tmp[i], tmp[j] = tmp[j], tmp[i]
	})
	return tmp
}

// RequestAndParsePlainIPResponse requests and parses a plain IP response.
func RequestAndParsePlainIPResponse(c *requests.HTTPClient, url string) (string, error) {
	req, err := requests.NewGetRequest(url, "", nil)
	if err != nil {
		return "", err
	}

	res, err := c.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	r, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	ipv4addr := net.ParseIP(strings.TrimSpace(string(r)))
	if ipv4addr == nil {
		return "", errors.New("could not parse ip response")
	}
	return ipv4addr.String(), err
}
