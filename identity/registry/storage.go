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

package registry

import (
	"fmt"
	"sync"
	"time"

	"github.com/mysteriumnetwork/node/identity"
	"github.com/pkg/errors"
)

const registrationStatusBucket = "registry_statuses"

type persistentStorage interface {
	Store(bucket string, data interface{}) error
	GetOneByField(bucket string, fieldName string, key interface{}, to interface{}) error
	GetAllFrom(bucket string, data interface{}) error
}

var errBoltNotFound = "not found"

// ErrNotFound represents an error where no info could be found in storage
var ErrNotFound = errors.New("no info for provided identity available in storage")

// RegistrationStatusStorage allows for storing of registration statuses.
type RegistrationStatusStorage struct {
	lock sync.Mutex
	bolt persistentStorage
}

type storedRegistrationStatus struct {
	ID string `storm:"id"`
	StoredRegistrationStatus
}

// NewRegistrationStatusStorage returns a new instance of the registration status storage
func NewRegistrationStatusStorage(bolt persistentStorage) *RegistrationStatusStorage {
	return &RegistrationStatusStorage{
		bolt: bolt,
	}
}

// Store stores the given promise for the given hermes.
func (rss *RegistrationStatusStorage) Store(status StoredRegistrationStatus) error {
	rss.lock.Lock()
	defer rss.lock.Unlock()

	s, err := rss.get(status.ChainID, status.Identity)
	if err == ErrNotFound {
		return rss.store(status)
	} else if err != nil {
		return err
	}

	switch s.RegistrationStatus {
	// can not be overridden
	case Registered:
		return nil
	default:
		s.RegistrationStatus = status.RegistrationStatus
	}

	return rss.store(s)
}

func (rss *RegistrationStatusStorage) store(status StoredRegistrationStatus) error {
	status.UpdatedAt = time.Now().UTC()
	store := &storedRegistrationStatus{
		ID:                       rss.makeKey(status.Identity, status.ChainID),
		StoredRegistrationStatus: status,
	}

	err := rss.bolt.Store(registrationStatusBucket, store)
	return errors.Wrap(err, "could not store registration status")
}

func (rss *RegistrationStatusStorage) get(chainID int64, identity identity.Identity) (StoredRegistrationStatus, error) {
	result := &storedRegistrationStatus{}
	err := rss.bolt.GetOneByField(registrationStatusBucket, "ID", rss.makeKey(identity, chainID), result)
	if err != nil {
		if err.Error() == errBoltNotFound {
			return StoredRegistrationStatus{}, ErrNotFound
		}
		return StoredRegistrationStatus{}, errors.Wrap(err, "could not get registration status")
	}
	return result.StoredRegistrationStatus, err
}

// Get fetches the promise for the given hermes.
func (rss *RegistrationStatusStorage) Get(chainID int64, identity identity.Identity) (StoredRegistrationStatus, error) {
	rss.lock.Lock()
	defer rss.lock.Unlock()
	return rss.get(chainID, identity)
}

// GetAll fetches all the registration statuses
func (rss *RegistrationStatusStorage) GetAll() ([]StoredRegistrationStatus, error) {
	rss.lock.Lock()
	defer rss.lock.Unlock()

	list := []storedRegistrationStatus{}
	err := rss.bolt.GetAllFrom(registrationStatusBucket, &list)
	if err != nil {
		if err.Error() == errBoltNotFound {
			return nil, ErrNotFound
		}
		return nil, errors.Wrap(err, "could not get all registration statuses")
	}

	result := make([]StoredRegistrationStatus, len(list))
	for i, l := range list {
		result[i] = l.StoredRegistrationStatus
	}
	return result, nil
}

func (rss *RegistrationStatusStorage) makeKey(identity identity.Identity, chainID int64) string {
	return fmt.Sprintf("%s|%d", identity.Address, chainID)
}
