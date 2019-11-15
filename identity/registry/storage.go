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
	Update(bucket string, object interface{}) error
}

var errBoltNotFound = "not found"

// ErrNotFound represents an error where no info could be found in storage
var ErrNotFound = errors.New("no info for provided identity available in storage")

// RegistrationStatusStorage allows for storing of registration statuses.
type RegistrationStatusStorage struct {
	lock sync.Mutex
	bolt persistentStorage
}

// NewRegistrationStatusStorage returns a new instance of the registration status storage
func NewRegistrationStatusStorage(bolt persistentStorage) *RegistrationStatusStorage {
	return &RegistrationStatusStorage{
		bolt: bolt,
	}
}

// Store stores the given promise for the given accountant.
func (rss *RegistrationStatusStorage) Store(status StoredRegistrationStatus) error {
	rss.lock.Lock()
	defer rss.lock.Unlock()

	s, err := rss.get(status.Identity)
	if err == ErrNotFound {
		return rss.store(status)
	} else if err != nil {
		return err
	}

	switch s.RegistrationStatus {
	// can only be overriden by registeredProvider and promotion
	case RegisteredConsumer:
		if status.RegistrationStatus == RegisteredProvider || status.RegistrationStatus == Promoting {
			s.RegistrationStatus = status.RegistrationStatus
			return rss.store(s)
		}
		return nil
	// can not be overriden
	case RegisteredProvider:
		return nil
	// can only be overriden by registered Provider
	case Promoting:
		if status.RegistrationStatus == RegisteredProvider {
			s.RegistrationStatus = status.RegistrationStatus
			return rss.store(s)
		}
		return nil
	default:
		s.RegistrationStatus = status.RegistrationStatus
	}

	return rss.store(s)
}

func (rss *RegistrationStatusStorage) store(status StoredRegistrationStatus) error {
	status.UpdatedAt = time.Now().UTC()
	return errors.Wrap(rss.bolt.Store(registrationStatusBucket, &status), "could not store registration status")
}

func (rss *RegistrationStatusStorage) get(identity identity.Identity) (StoredRegistrationStatus, error) {
	result := &StoredRegistrationStatus{}
	err := rss.bolt.GetOneByField(registrationStatusBucket, "Identity", identity, result)
	if err != nil {
		if err.Error() == errBoltNotFound {
			err = ErrNotFound
		} else {
			err = errors.Wrap(err, "could not get registration status")
		}
	}
	return *result, err
}

// Get fetches the promise for the given accountant.
func (rss *RegistrationStatusStorage) Get(identity identity.Identity) (StoredRegistrationStatus, error) {
	rss.lock.Lock()
	defer rss.lock.Unlock()
	return rss.get(identity)
}

// GetAll fetches all the registration statuses
func (rss *RegistrationStatusStorage) GetAll() ([]StoredRegistrationStatus, error) {
	rss.lock.Lock()
	defer rss.lock.Unlock()

	res := []StoredRegistrationStatus{}
	err := rss.bolt.GetAllFrom(registrationStatusBucket, &res)
	if err != nil {
		if err.Error() == errBoltNotFound {
			err = ErrNotFound
		} else {
			err = errors.Wrap(err, "could not get all registration statuses")
		}
	}
	return res, err
}
