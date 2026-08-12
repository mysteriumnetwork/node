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
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"

	runtime_service "github.com/mysteriumnetwork/node/services/runtime"
)

const (
	// servicesPath is the registry listing endpoint.
	servicesPath = "/v1/service"
	// requestTimeout bounds a single registry fetch. A create request waits for
	// it, so it stays well below any operator-facing timeout.
	requestTimeout = 15 * time.Second
	// maxResponseSize caps what a registry response can cost this node.
	maxResponseSize = 4 << 20
	// defaultCacheTTL keeps repeated creates off the network without letting a
	// node act on a stale allow list for long.
	defaultCacheTTL = time.Minute
)

// Client reads the service registry over HTTP.
type Client struct {
	endpoint string
	http     *http.Client
	ttl      time.Duration
	now      func() time.Time

	mu        sync.Mutex
	services  map[string][]Service
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

// Lookup returns the registry entry that installs as the given service name.
// It answers ErrNotListed for anything the registry does not publish, which is
// what keeps a create request from installing an arbitrary workload.
func (client *Client) Lookup(serviceName string) (Service, error) {
	serviceType := runtime_service.NormalizeServiceType(serviceName)
	if serviceType == "" {
		return Service{}, errors.New("runtime service name is required")
	}

	services, cached, err := client.snapshot(false)
	if err != nil {
		return Service{}, err
	}
	entry, found, err := selectService(services, serviceType)
	if err != nil {
		return Service{}, err
	}
	if !found && cached {
		// A name missing from a cached listing may simply be newer than the
		// cache, so a miss is confirmed against the registry itself before a
		// legitimate service is refused.
		if services, _, err = client.snapshot(true); err != nil {
			return Service{}, err
		}
		if entry, found, err = selectService(services, serviceType); err != nil {
			return Service{}, err
		}
	}
	if !found {
		return Service{}, errors.Wrapf(ErrNotListed, "runtime service %q", serviceType)
	}

	if err := entry.validate(hostOS, hostArchitecture); err != nil {
		return Service{}, err
	}
	return entry, nil
}

func selectService(services map[string][]Service, serviceType string) (Service, bool, error) {
	entries := services[serviceType]
	switch len(entries) {
	case 0:
		return Service{}, false, nil
	case 1:
		return entries[0], true, nil
	default:
		// Two entries claiming one service type make the approved workload
		// ambiguous, and guessing between them is exactly what this gate exists
		// to prevent.
		return Service{}, false, errors.Errorf(
			"runtime service registry lists %d conflicting entries for %q", len(entries), serviceType,
		)
	}
}

// snapshot returns the current listing and reports whether it was served from
// cache without touching the registry.
func (client *Client) snapshot(force bool) (map[string][]Service, bool, error) {
	client.mu.Lock()
	cached := client.services
	fresh := client.now().Sub(client.fetchedAt) < client.ttl
	client.mu.Unlock()

	if !force && cached != nil && fresh {
		return cached, true, nil
	}

	services, err := client.fetch()
	if err != nil {
		return nil, false, err
	}

	client.mu.Lock()
	client.services = services
	client.fetchedAt = client.now()
	client.mu.Unlock()

	return services, false, nil
}

func (client *Client) fetch() (map[string][]Service, error) {
	response, err := client.http.Get(client.endpoint)
	if err != nil {
		return nil, errors.Wrap(err, "failed to reach runtime service registry")
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, errors.Errorf("runtime service registry returned status %d", response.StatusCode)
	}

	services, err := decodeListing(io.LimitReader(response.Body, maxResponseSize+1))
	if err != nil {
		return nil, err
	}

	return indexServices(services), nil
}

// decodeListing reads the listing envelope the registry serves.
func decodeListing(body io.Reader) ([]Service, error) {
	var envelope struct {
		Services []Service `json:"services"`
	}
	if err := json.NewDecoder(body).Decode(&envelope); err != nil {
		return nil, errors.Wrap(err, "failed to read runtime service registry listing")
	}
	return envelope.Services, nil
}

func indexServices(services []Service) map[string][]Service {
	indexed := make(map[string][]Service, len(services))
	for _, service := range services {
		serviceType := service.ServiceType()
		if serviceType == "" {
			// Kept out of the index rather than rejected outright: an entry the
			// node cannot address at all must not break lookups of the rest.
			continue
		}
		indexed[serviceType] = append(indexed[serviceType], service)
	}
	return indexed
}
