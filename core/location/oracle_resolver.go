/*
 * Copyright (C) 2019 The "MysteriumNetwork/node" Authors.
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

package location

import (
	"context"
	"fmt"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/mysteriumnetwork/node/requests"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

type oracleResolver struct {
	httpClient *requests.HTTPClient
	address    string
}

// NewOracleResolver returns new db resolver initialized from Location Oracle service
func NewOracleResolver(httpClient *requests.HTTPClient, address string) *oracleResolver {
	return &oracleResolver{
		httpClient: httpClient,
		address:    address,
	}
}

// DetectLocation detects current IP-address provides location information for the IP.
func (o *oracleResolver) DetectLocation() (Location, error) {
	log.Debug().Msg("Detecting with oracle resolver")

	var boff backoff.BackOff
	eback := backoff.NewExponentialBackOff()
	eback.MaxElapsedTime = time.Second * 20
	eback.InitialInterval = time.Second * 2
	boff = backoff.WithMaxRetries(eback, 10)

	var loc Location
	retry := func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		request, err := requests.NewGetRequestWithContext(ctx, o.address, "", nil)
		if err != nil {
			return errors.Wrap(err, "failed to create request")
		}
		if err := o.httpClient.DoRequestAndParseResponse(request, &loc); err != nil {
			log.Err(err).Msg("Location detection failed, will try again")
			return err
		}
		return nil
	}

	if err := backoff.Retry(retry, boff); err != nil {
		return Location{}, fmt.Errorf("could not detect location: %w", err)
	}
	return loc, nil
}
