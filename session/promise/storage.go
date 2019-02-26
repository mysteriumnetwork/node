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

package promise

import (
	"errors"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/mysteriumnetwork/node/identity"
)

const promiseBucketPrefix = "stored-promise-"
const firstPromiseID = uint64(1)

var (
	// ErrPromiseNotFound represents the error we return when trying to update a non existing promise
	ErrPromiseNotFound = errors.New("promise not found")
	// errBoltNotFound represents the bolts not found error
	errBoltNotFound = errors.New("not found")
)

// Storer allows us to get all promises, save and update them
type Storer interface {
	Store(bucket string, object interface{}) error
	GetAllFrom(bucket string, array interface{}) error
	Update(bucket string, object interface{}) error
	GetOneByField(bucket string, fieldName string, key interface{}, to interface{}) error
	GetLast(bucket string, to interface{}) error
	GetBuckets() []string
}

// Storage stores promises. It also issues sequence ID's for promises.
// It's designed to be used as a singleton for promise storage.
type Storage struct {
	storage Storer
	sync.Mutex
}

// NewStorage returns a new instance of promise storage
func NewStorage(storage Storer) *Storage {
	return &Storage{
		storage: storage,
	}
}

// StoredPromise is a representation of a promise in storage. It stores the message that the consumer sent.
type StoredPromise struct {
	SequenceID       uint64 `storm:"id"`
	Message          *Message
	AddedAt          time.Time
	UpdatedAt        time.Time
	UnconsumedAmount uint64
	ConsumerID       identity.Identity
	Receiver         identity.Identity
	Cleared          bool
}

// GetNewSeqIDForIssuer returns a new sequenceID for the provided issuer.
// The operation is atomic and thread safe.
func (s *Storage) GetNewSeqIDForIssuer(consumerID, receiverID, issuerID identity.Identity) (uint64, error) {
	s.Lock()
	defer s.Unlock()
	lastPromise, err := s.getLastPromise(issuerID)
	if err != nil && err.Error() == errBoltNotFound.Error() {
		// we do not have a previous history with the issuer, ask for a promise no1, store it
		err := s.store(issuerID, StoredPromise{
			SequenceID: firstPromiseID,
			ConsumerID: consumerID,
			Receiver:   receiverID,
		})
		return firstPromiseID, err
	} else if err != nil {
		return 0, err
	}

	newID := lastPromise.SequenceID + 1
	err = s.store(issuerID, StoredPromise{
		SequenceID: newID,
		ConsumerID: consumerID,
		Receiver:   receiverID,
	})
	return newID, err
}

// Store allows for storing of arbitrary promise.
func (s *Storage) Store(issuerID identity.Identity, sp StoredPromise) error {
	s.Lock()
	defer s.Unlock()
	return s.store(issuerID, sp)
}

// Update updates a promise in the DB
func (s *Storage) Update(issuerID identity.Identity, sp StoredPromise) error {
	s.Lock()
	defer s.Unlock()

	// The storage layers update doesn't really care if the promise exists, it will just insert a new one.
	// In this case - we'll want to make sure we don't update something that does not exist
	_, err := s.getPromiseByID(issuerID, sp.SequenceID)
	if err != nil {
		return err
	}

	return s.update(issuerID, sp)
}

// GetLastPromise fetches the last promise for the provider
func (s *Storage) GetLastPromise(issuerID identity.Identity) (StoredPromise, error) {
	s.Lock()
	defer s.Unlock()
	promise, err := s.getLastPromise(issuerID)
	return promise, err
}

// GetAllKnownIssuers returns a list of known issuer addresses
func (s *Storage) GetAllKnownIssuers() []identity.Identity {
	s.Lock()
	defer s.Unlock()

	buckets := s.storage.GetBuckets()

	res := make([]identity.Identity, 0)
	for i := range buckets {
		if strings.HasPrefix(buckets[i], promiseBucketPrefix) {
			res = append(res, identity.FromAddress(strings.TrimPrefix(buckets[i], promiseBucketPrefix)))
		}
	}

	return res
}

var errNoPromiseForConsumer = errors.New("no promise for consumer")

// FindPromiseForConsumer returns the last known promise for the issuer/consumer combo
// It checks if any promises match, and if they do checks to see if any later promises have been cleared.FindPromiseForConsumer
func (s *Storage) FindPromiseForConsumer(issuerID, consumerID identity.Identity) (StoredPromise, error) {
	s.Lock()
	defer s.Unlock()
	promises, err := s.getAllPromisesForIssuer(issuerID)
	if err != nil {
		return StoredPromise{}, err
	}

	sortPromisesDesc(promises)

	// Iterate from the last promise to the first
	for i := 0; i < len(promises); i++ {
		// if we find a cleared promise, it means we've done our job here - we'll need to issue a new id
		if promises[i].Cleared {
			return StoredPromise{}, errNoPromiseForConsumer
		}
		// otherwise, we're free to extend
		if promises[i].ConsumerID == consumerID {
			return promises[i], nil
		}
	}

	return StoredPromise{}, errNoPromiseForConsumer
}

func sortPromisesDesc(promises []StoredPromise) {
	// sort by sequenceID, descending
	sort.Slice(promises, func(i, j int) bool {
		return promises[i].SequenceID > promises[j].SequenceID
	})
}

// GetAllPromisesFromIssuer fetches all the promises known for the given issuer
func (s *Storage) GetAllPromisesFromIssuer(issuerID identity.Identity) ([]StoredPromise, error) {
	s.Lock()
	defer s.Unlock()
	return s.getAllPromisesForIssuer(issuerID)
}

func (s *Storage) getPromiseByID(issuerID identity.Identity, sequenceID uint64) (StoredPromise, error) {
	var sp StoredPromise
	err := s.storage.GetOneByField(getBucketNameFromIssuer(issuerID), "SequenceID", sequenceID, &sp)
	return sp, err
}

func (s *Storage) getLastPromise(issuerID identity.Identity) (StoredPromise, error) {
	var sp StoredPromise
	err := s.storage.GetLast(getBucketNameFromIssuer(issuerID), &sp)
	return sp, err
}

func (s *Storage) getAllPromisesForIssuer(issuerID identity.Identity) (res []StoredPromise, err error) {
	err = s.storage.GetAllFrom(getBucketNameFromIssuer(issuerID), &res)
	return
}

func (s *Storage) update(issuerID identity.Identity, promise StoredPromise) error {
	promise.UpdatedAt = time.Now().UTC()
	return s.storage.Update(getBucketNameFromIssuer(issuerID), &promise)
}

func (s *Storage) store(issuerID identity.Identity, sp StoredPromise) error {
	sp.AddedAt = time.Now().UTC()
	return s.storage.Store(getBucketNameFromIssuer(issuerID), &sp)
}

func getBucketNameFromIssuer(issuerID identity.Identity) string {
	return promiseBucketPrefix + issuerID.Address
}
