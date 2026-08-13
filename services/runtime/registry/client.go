/*
 * Copyright (C) 2026 The "MysteriumNetwork/node" Authors.
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

package registry

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
)

const (
	// servicesPath is the registry listing endpoint.
	servicesPath = "/v1/service"
	// requestTimeout bounds a single registry fetch. A create request waits for
	// it, so it stays well below any operator-facing timeout.
	requestTimeout = 15 * time.Second
	// maxResponseSize caps what a registry response can cost this node.
	maxResponseSize = 4 << 20
	// defaultCacheTTL keeps repeated consumers and retries off the network
	// without letting a node act on a stale allow list for long.
	defaultCacheTTL = time.Minute
)

// Client reads the service registry over HTTP.
type Client struct {
	endpoint string
	http     *http.Client
	ttl      time.Duration
	now      func() time.Time

	mu        sync.Mutex
	services  []Service
	fetchErr  error
	fetchedAt time.Time
}

// NewClient creates a registry client for the given base address. The address
// is the registry root; the client appends the listing path itself so a
// misconfigured path cannot silently point the node at another document.
func NewClient(address string) (*Client, error) {
	endpoint, err := servicesEndpoint(address)
	if err != nil {
		return nil, err
	}

	return &Client{
		endpoint: endpoint,
		http:     &http.Client{Timeout: requestTimeout},
		ttl:      defaultCacheTTL,
		now:      time.Now,
	}, nil
}

func servicesEndpoint(address string) (string, error) {
	trimmed := strings.TrimSpace(address)
	if trimmed == "" {
		return "", errors.New("runtime service registry address is required")
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return "", errors.Wrap(err, "invalid runtime service registry address")
	}
	if parsed.Host == "" {
		return "", errors.New("runtime service registry address must include a host")
	}
	// The listing decides which workloads this node runs, so whoever can rewrite
	// it in flight chooses what executes here; digest pinning does not help,
	// because the same attacker supplies the digest.
	if parsed.Scheme != "https" {
		return "", errors.Errorf("runtime service registry address must be https, got %q", parsed.Scheme)
	}

	parsed.Path = strings.TrimSuffix(parsed.Path, "/") + servicesPath
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String(), nil
}

// ListServices returns one cached snapshot of the definitions the registry
// publishes. Consumers select and validate entries from this same snapshot, so
// installation and reconciliation agree on both names and artifact digests.
func (client *Client) ListServices() ([]Service, error) {
	services, err := client.snapshot()
	if err != nil {
		return nil, err
	}
	return append([]Service(nil), services...), nil
}

// snapshot returns the current listing. The lock intentionally covers the
// fetch: concurrent callers share one request instead of stampeding the
// registry when the cache expires.
func (client *Client) snapshot() ([]Service, error) {
	client.mu.Lock()
	defer client.mu.Unlock()

	if !client.fetchedAt.IsZero() && client.now().Sub(client.fetchedAt) < client.ttl {
		return client.services, client.fetchErr
	}

	services, err := client.fetch()
	client.fetchedAt = client.now()
	if err != nil {
		// Cache the failure, not the previous listing. Callers must never act on
		// stale approval data, but they also should not hammer an unhealthy
		// registry until this short retry window expires.
		client.fetchErr = err
		return nil, err
	}

	client.services = services
	client.fetchErr = nil
	return services, nil
}

func (client *Client) fetch() ([]Service, error) {
	response, err := client.http.Get(client.endpoint)
	if err != nil {
		return nil, errors.Wrap(err, "failed to reach runtime service registry")
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, errors.Errorf("runtime service registry returned status %d", response.StatusCode)
	}

	services, err := decodeListing(response.Body)
	if err != nil {
		return nil, err
	}

	return services, nil
}

// decodeListing reads the listing envelope the registry serves.
func decodeListing(body io.Reader) ([]Service, error) {
	contents, err := io.ReadAll(io.LimitReader(body, maxResponseSize+1))
	if err != nil {
		return nil, errors.Wrap(err, "failed to read runtime service registry listing")
	}
	if len(contents) > maxResponseSize {
		return nil, errors.Errorf("runtime service registry listing exceeds %d bytes", maxResponseSize)
	}

	var envelope struct {
		// RawMessage distinguishes a deliberately empty array from a missing or
		// null field. Only the former is an authoritative statement that no
		// workloads are currently approved.
		Services json.RawMessage `json:"services"`
	}
	decoder := json.NewDecoder(bytes.NewReader(contents))
	if err := decoder.Decode(&envelope); err != nil {
		return nil, errors.Wrap(err, "failed to read runtime service registry listing")
	}

	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return nil, errors.New("runtime service registry listing contains more than one JSON document")
		}
		return nil, errors.Wrap(err, "runtime service registry listing contains trailing data")
	}

	servicesJSON := bytes.TrimSpace(envelope.Services)
	if len(servicesJSON) == 0 || bytes.Equal(servicesJSON, []byte("null")) {
		return nil, errors.New("runtime service registry listing must contain a non-null services array")
	}

	var services []Service
	if err := json.Unmarshal(servicesJSON, &services); err != nil {
		return nil, errors.Wrap(err, "failed to read runtime service registry services")
	}
	return services, nil
}
