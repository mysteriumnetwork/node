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
	"fmt"
	"reflect"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/mysteriumnetwork/node/identity"
	"github.com/stretchr/testify/assert"
)

var (
	consumerID = identity.FromAddress("0x0")
	issuerID   = identity.FromAddress("0x00")
	receiverID = identity.FromAddress("0x000")
	timeMock   = time.Now()
	mock       = map[string][]StoredPromise{
		getBucketNameFromIssuer(issuerID): {
			{
				SequenceID: 1,
				AddedAt:    timeMock,
				UpdatedAt:  timeMock,
				ConsumerID: consumerID,
				Receiver:   receiverID,
			},
		},
	}
)

func Test_Storage_IssuesOneForUnknownID(t *testing.T) {
	ms := newMockStorage(nil)
	s := NewStorage(ms)
	res, err := s.GetNewSeqIDForIssuer(consumerID, receiverID, issuerID)
	assert.Nil(t, err)
	assert.Equal(t, firstPromiseID, res)

	var stored []StoredPromise
	err = ms.GetAllFrom(getBucketNameFromIssuer(issuerID), &stored)
	assert.Nil(t, err)
	assert.Equal(t, firstPromiseID, stored[0].SequenceID)
	assert.Nil(t, stored[0].Message)
	assert.Equal(t, consumerID, stored[0].ConsumerID)
	assert.Equal(t, receiverID, stored[0].Receiver)
	assert.False(t, stored[0].AddedAt.IsZero())
	assert.True(t, stored[0].UpdatedAt.IsZero())
}

func Test_Storage_UpdatesPromise(t *testing.T) {
	ms := newMockStorage(&mock)
	s := NewStorage(ms)

	var stored []StoredPromise
	err := ms.GetAllFrom(getBucketNameFromIssuer(issuerID), &stored)
	assert.Nil(t, err)
	assert.Nil(t, stored[0].Message)

	msg := &Message{
		Amount: 1,
	}

	err = s.Update(issuerID, StoredPromise{
		SequenceID: 1,
		Message:    msg,
	})
	assert.Nil(t, err)

	promise, err := s.getPromiseByID(issuerID, 1)
	assert.Nil(t, err)
	assert.Equal(t, msg, promise.Message)
}

func Test_Storage_UpdateErrsOnNonExistingPromise(t *testing.T) {
	ms := newMockStorage(nil)
	s := NewStorage(ms)
	err := s.Update(issuerID, StoredPromise{
		SequenceID: 1,
	})
	assert.Equal(t, errNotFound, err)
}

func Test_Storage_Store(t *testing.T) {
	ms := newMockStorage(nil)
	s := NewStorage(ms)

	mockMsg := Message{}
	err := s.Store(issuerID, StoredPromise{
		SequenceID: 1,
		Message:    &mockMsg,
	})
	assert.Nil(t, err)

	lp, err := s.getLastPromise(issuerID)
	assert.Nil(t, err)
	assert.Equal(t, uint64(1), lp.SequenceID)
	assert.Equal(t, mockMsg, *lp.Message)
	assert.False(t, lp.AddedAt.IsZero())
	assert.True(t, lp.UpdatedAt.IsZero())
}

func Test_Storage_IssuesSecondForKnownID(t *testing.T) {
	ms := newMockStorage(&mock)
	s := NewStorage(ms)
	res, err := s.GetNewSeqIDForIssuer(consumerID, receiverID, issuerID)
	assert.Nil(t, err)
	assert.Equal(t, uint64(2), res)
}

func Test_Storage_GetAllKnownIssuers_GetsKnownIdentities(t *testing.T) {
	ms := newMockStorage(&mock)
	s := NewStorage(ms)
	id2 := identity.FromAddress("0x1")
	err := s.Store(id2, StoredPromise{
		SequenceID: 1,
	})
	assert.Nil(t, err)

	issuers := s.GetAllKnownIssuers()
	assert.Len(t, issuers, 2)

	assert.True(t, containsIdentity(issuers, issuerID))
	assert.True(t, containsIdentity(issuers, id2))
}

func containsIdentity(s []identity.Identity, e identity.Identity) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func Test_Storage_GetAllKnownIssuers_ReturnsEmptyIdentityList(t *testing.T) {
	ms := newMockStorage(nil)
	s := NewStorage(ms)
	issuers := s.GetAllKnownIssuers()
	assert.Len(t, issuers, 0)
}

func Test_Storage_GetAllPromisesForIssuer_ReturnsEmptyPromiseList(t *testing.T) {
	ms := newMockStorage(nil)
	s := NewStorage(ms)
	res, err := s.GetAllPromisesFromIssuer(issuerID)
	assert.Nil(t, err)
	assert.Len(t, res, 0)
}

func Test_LoadPaymentInfo_EmptyOnError(t *testing.T) {
	ms := newMockStorage(nil)
	s := NewStorage(ms)
	pi := s.LoadPaymentInfo(issuerID, receiverID, consumerID)
	assert.Equal(t, &PaymentInfo{
		LastPromise: LastPromise{
			SequenceID: 1,
			Amount:     0,
		},
	}, pi)
}

func Test_LoadPaymentInfo_ZeroAmountOnNoMessage(t *testing.T) {
	ms := newMockStorage(&mock)
	s := NewStorage(ms)
	pi := s.LoadPaymentInfo(issuerID, receiverID, consumerID)
	assert.Equal(t, &PaymentInfo{
		LastPromise: LastPromise{
			SequenceID: 1,
			Amount:     0,
		},
		FreeCredit: 0,
	}, pi)
}

func Test_LoadPaymentInfo_LoadsProperInfo(t *testing.T) {
	newMock := map[string][]StoredPromise{
		getBucketNameFromIssuer(issuerID): {
			{
				SequenceID: 10,
				AddedAt:    timeMock,
				UpdatedAt:  timeMock,
				Message: &Message{
					Amount: 300,
				},
				UnconsumedAmount: 200,
				ConsumerID:       consumerID,
				Receiver:         receiverID,
			},
		},
	}
	ms := newMockStorage(&newMock)
	s := NewStorage(ms)
	pi := s.LoadPaymentInfo(consumerID, receiverID, issuerID)
	assert.Equal(t, uint64(10), pi.LastPromise.SequenceID)
	assert.Equal(t, uint64(300), pi.LastPromise.Amount)
	assert.Equal(t, uint64(200), pi.FreeCredit)
}

func Test_LoadPaymentInfo_WithPreviousConsumersIssuesNewID(t *testing.T) {
	newMock := map[string][]StoredPromise{
		getBucketNameFromIssuer(issuerID): {
			{
				SequenceID: 10,
				AddedAt:    timeMock,
				UpdatedAt:  timeMock,
				Message: &Message{
					Amount: 300,
				},
				UnconsumedAmount: 200,
				ConsumerID:       receiverID,
				Receiver:         receiverID,
			},
			{
				SequenceID: 11,
				AddedAt:    timeMock,
				UpdatedAt:  timeMock,
				Message: &Message{
					Amount: 300,
				},
				UnconsumedAmount: 200,
				ConsumerID:       issuerID,
				Receiver:         receiverID,
			},
		},
	}

	ms := newMockStorage(&newMock)
	s := NewStorage(ms)
	pi := s.LoadPaymentInfo(consumerID, receiverID, issuerID)
	assert.Equal(t, &PaymentInfo{
		LastPromise: LastPromise{
			SequenceID: 12,
			Amount:     0,
		},
	}, pi)
}

func Test_LoadPaymentInfo_NextIfReceiversDontMatch(t *testing.T) {
	newMock := map[string][]StoredPromise{
		getBucketNameFromIssuer(issuerID): {
			{
				SequenceID: 10,
				AddedAt:    timeMock,
				UpdatedAt:  timeMock,
				Message: &Message{
					Amount: 300,
				},
				UnconsumedAmount: 200,
				ConsumerID:       consumerID,
				Receiver:         issuerID,
			},
		},
	}

	ms := newMockStorage(&newMock)
	s := NewStorage(ms)
	pi := s.LoadPaymentInfo(consumerID, receiverID, issuerID)
	assert.Equal(t, &PaymentInfo{
		LastPromise: LastPromise{
			SequenceID: 11,
			Amount:     0,
		},
	}, pi)
}

func Test_Storage_GetAllPromisesForIssuer_GetsAllPromises(t *testing.T) {
	ms := newMockStorage(&mock)
	s := NewStorage(ms)
	res, err := s.GetAllPromisesFromIssuer(issuerID)
	assert.Nil(t, err)
	assert.Len(t, res, 1)
}

func Test_Storage_IssuesUniqueSequencesForMultipleIssuers(t *testing.T) {
	ms := newMockStorage(&mock)
	s := NewStorage(ms)
	res, err := s.GetNewSeqIDForIssuer(consumerID, receiverID, issuerID)
	assert.Nil(t, err)
	assert.Equal(t, uint64(2), res)

	nextID := identity.FromAddress("0x1")
	res, err = s.GetNewSeqIDForIssuer(consumerID, receiverID, nextID)
	assert.Nil(t, err)
	assert.Equal(t, firstPromiseID, res)
}

func Test_Storage_DoesAtomicIncrementsUnderConcurrentLoad(t *testing.T) {
	ms := newMockStorage(nil)
	s := NewStorage(ms)

	routineCount := 1000

	tracker := make([]uint64, routineCount)

	wg := sync.WaitGroup{}
	for i := 0; i < routineCount; i++ {
		wg.Add(1)
		go func(j int) {
			defer wg.Done()
			seqID, err := s.GetNewSeqIDForIssuer(consumerID, receiverID, issuerID)
			assert.Nil(t, err)
			tracker[j] = seqID
		}(i)
	}

	wg.Wait()

	// order the slice
	sort.Slice(tracker, func(i, j int) bool { return tracker[i] < tracker[j] })

	// we should contain the ints in range [1;1000]
	for i := 0; i < routineCount; i++ {
		assert.Equal(t, uint64(i+1), tracker[i])
	}
}

func Test_FindPromiseForConsumer_ErrsOnEmptySlice(t *testing.T) {
	ms := newMockStorage(nil)
	s := NewStorage(ms)
	_, err := s.FindPromiseForConsumer(issuerID, receiverID, identity.FromAddress("0x000"))
	assert.Equal(t, errNoPromiseForConsumer, err)
}

func Test_FindPromiseForConsumer_ErrsOnNoConsumerPromise(t *testing.T) {
	mock := map[string][]StoredPromise{
		getBucketNameFromIssuer(issuerID): {
			{
				SequenceID: 1,
				AddedAt:    timeMock,
				UpdatedAt:  timeMock,
			},
			{
				SequenceID: 2,
				AddedAt:    timeMock,
				UpdatedAt:  timeMock,
			},
		},
	}
	ms := newMockStorage(&mock)
	s := NewStorage(ms)
	_, err := s.FindPromiseForConsumer(identity.FromAddress("0x000"), receiverID, issuerID)
	assert.Equal(t, errNoPromiseForConsumer, err)
}

func Test_FindPromiseForConsumer_FindsConsumer(t *testing.T) {
	consumerID := identity.FromAddress("0x000")
	mock := map[string][]StoredPromise{
		getBucketNameFromIssuer(issuerID): {
			{
				SequenceID: 1,
				AddedAt:    timeMock,
				UpdatedAt:  timeMock,
				ConsumerID: consumerID,
				Receiver:   receiverID,
			},
			{
				SequenceID: 2,
				AddedAt:    timeMock,
				UpdatedAt:  timeMock,
			},
		},
	}
	ms := newMockStorage(&mock)
	s := NewStorage(ms)
	pr, err := s.FindPromiseForConsumer(consumerID, receiverID, issuerID)
	assert.Nil(t, err)
	assert.Equal(t, uint64(1), pr.SequenceID)
	assert.Equal(t, consumerID, pr.ConsumerID)
}

func Test_FindPromiseForConsumer_FindsClearerd(t *testing.T) {
	consumerID := identity.FromAddress("0x000")
	mock := map[string][]StoredPromise{
		getBucketNameFromIssuer(issuerID): {
			{
				SequenceID: 1,
				AddedAt:    timeMock,
				UpdatedAt:  timeMock,
				ConsumerID: consumerID,
			},
			{
				SequenceID: 2,
				AddedAt:    timeMock,
				UpdatedAt:  timeMock,
				Cleared:    true,
			},
		},
	}
	ms := newMockStorage(&mock)
	s := NewStorage(ms)
	_, err := s.FindPromiseForConsumer(issuerID, receiverID, consumerID)
	assert.Equal(t, errNoPromiseForConsumer, err)
}

func Test_MockStorage(t *testing.T) {
	ms := newMockStorage(nil)

	bucket := "test"

	err := ms.Store(bucket, time.Now())
	assert.Equal(t, "mockStorage.Store expects *StoredPromise but got time.Time instead", err.Error())

	err = ms.GetAllFrom(bucket, time.Now())
	assert.Equal(t, "mockStorage.Store expects *[]StoredPromise but got time.Time instead", err.Error())

	err = ms.Update(bucket, time.Now())
	assert.Equal(t, "mockStorage.Update expects *StoredPromise but got time.Time instead", err.Error())

	err = ms.GetOneByField(bucket, "test", "test", time.Now())
	assert.Equal(t, "mockStorage.GetOneByField expects *StoredPromise but got time.Time instead", err.Error())

	err = ms.GetOneByField(bucket, "test", "test", &StoredPromise{})
	assert.Equal(t, "mockStorage.GetOneByField expects a fieldname of SequenceID but got test instead", err.Error())

	err = ms.GetLast(bucket, time.Now())
	assert.Equal(t, "mockStorage.GetLast expects *StoredPromise but got time.Time instead", err.Error())

	var res []StoredPromise
	err = ms.GetAllFrom(bucket, &res)
	assert.Nil(t, err)
	assert.Len(t, res, 0)

	theNonExistingOne := StoredPromise{}
	err = ms.GetLast(bucket, &theNonExistingOne)
	assert.Equal(t, errNotFound, err)

	toInsert := &StoredPromise{
		SequenceID: 1,
		Message:    nil,
		AddedAt:    time.Now(),
		UpdatedAt:  time.Now(),
	}

	err = ms.Store(bucket, toInsert)
	assert.Nil(t, err)

	err = ms.GetAllFrom(bucket, &res)
	assert.Nil(t, err)
	assert.Len(t, res, 1)
	assert.Nil(t, res[0].Message)

	copy := res[0]
	copy.Message = &Message{Amount: 1}
	err = ms.Update(bucket, &copy)
	assert.Nil(t, err)

	res = []StoredPromise{}
	err = ms.GetAllFrom(bucket, &res)
	assert.Nil(t, err)
	assert.Len(t, res, 1)
	assert.Equal(t, copy.Message, res[0].Message)

	gottenOne := StoredPromise{}
	err = ms.GetOneByField(bucket, "SequenceID", toInsert.SequenceID, &gottenOne)
	assert.Nil(t, err)
	assert.Equal(t, copy.Message, gottenOne.Message)

	err = ms.GetOneByField(bucket, "SequenceID", "something", &gottenOne)
	assert.Equal(t, errNotFound, err)

	theLastOne := StoredPromise{}
	err = ms.GetLast(bucket, &theLastOne)
	assert.Nil(t, err)
	assert.Equal(t, copy.Message, theLastOne.Message)

	buckets := ms.GetBuckets()
	assert.Len(t, buckets, 1)
	assert.Equal(t, bucket, buckets[0])
}

type mockStorage struct {
	inMemStorage map[string][]StoredPromise
}

func newMockStorage(mocks *map[string][]StoredPromise) *mockStorage {
	if mocks == nil {
		return &mockStorage{
			inMemStorage: make(map[string][]StoredPromise),
		}
	}
	copied := make(map[string][]StoredPromise, len(*mocks))
	for k, v := range *mocks {
		values := make([]StoredPromise, len(v))
		copy(values, v)
		copied[k] = values
	}
	return &mockStorage{
		inMemStorage: copied,
	}
}

func (ms *mockStorage) Store(bucket string, object interface{}) error {
	sp, ok := object.(*StoredPromise)
	if !ok {
		providedType := reflect.TypeOf(object)
		return fmt.Errorf("mockStorage.Store expects *StoredPromise but got %v instead", providedType)
	}

	spToStore := *sp
	if _, ok := ms.inMemStorage[bucket]; ok {
		ms.inMemStorage[bucket] = append(ms.inMemStorage[bucket], spToStore)
	} else {
		ms.inMemStorage[bucket] = []StoredPromise{
			spToStore,
		}
	}

	return nil
}

func (ms *mockStorage) GetAllFrom(bucket string, array interface{}) error {
	casted, ok := array.(*[]StoredPromise)
	if !ok {
		providedType := reflect.TypeOf(array)
		return fmt.Errorf("mockStorage.Store expects *[]StoredPromise but got %v instead", providedType)
	}

	if _, ok := ms.inMemStorage[bucket]; ok {
		*casted = append(*casted, ms.inMemStorage[bucket]...)
	} else {
		*casted = []StoredPromise{}
	}

	return nil
}

func (ms *mockStorage) Update(bucket string, data interface{}) error {
	casted, ok := data.(*StoredPromise)
	if !ok {
		providedType := reflect.TypeOf(data)
		return fmt.Errorf("mockStorage.Update expects *StoredPromise but got %v instead", providedType)
	}
	if _, ok := ms.inMemStorage[bucket]; ok {
		for i := range ms.inMemStorage[bucket] {
			if ms.inMemStorage[bucket][i].SequenceID == casted.SequenceID {
				ms.inMemStorage[bucket][i].Message = casted.Message
				ms.inMemStorage[bucket][i].UpdatedAt = casted.UpdatedAt
				break
			}
		}
	}

	return nil
}

var errNotFound = errors.New("not found")

func (ms *mockStorage) GetOneByField(bucket string, fieldName string, key interface{}, to interface{}) error {
	casted, ok := to.(*StoredPromise)
	if !ok {
		providedType := reflect.TypeOf(to)
		return fmt.Errorf("mockStorage.GetOneByField expects *StoredPromise but got %v instead", providedType)
	}
	if fieldName != "SequenceID" {
		return fmt.Errorf("mockStorage.GetOneByField expects a fieldname of SequenceID but got %v instead", fieldName)
	}

	if _, ok := ms.inMemStorage[bucket]; ok {
		found := false
		for i := range ms.inMemStorage[bucket] {
			if ms.inMemStorage[bucket][i].SequenceID == key {
				found = true
				copy := ms.inMemStorage[bucket][i]
				*casted = copy
				break
			}
		}
		if !found {
			return errNotFound
		}
	} else {
		return errNotFound
	}

	return nil
}

func (ms *mockStorage) GetLast(bucket string, to interface{}) error {
	casted, ok := to.(*StoredPromise)
	if !ok {
		providedType := reflect.TypeOf(to)
		return fmt.Errorf("mockStorage.GetLast expects *StoredPromise but got %v instead", providedType)
	}
	if _, ok := ms.inMemStorage[bucket]; ok {
		copy := ms.inMemStorage[bucket][len(ms.inMemStorage[bucket])-1]
		*casted = copy
	} else {
		return errNotFound
	}

	return nil
}

func (ms *mockStorage) GetBuckets() []string {
	res := make([]string, len(ms.inMemStorage))
	i := 0
	for k := range ms.inMemStorage {
		res[i] = k
		i++
	}

	return res
}
