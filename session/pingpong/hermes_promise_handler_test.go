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

package pingpong

import (
	"errors"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/mysteriumnetwork/node/core/node/event"
	"github.com/mysteriumnetwork/node/eventbus"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/identity/registry"
	"github.com/mysteriumnetwork/payments/crypto"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
)

func TestHermesPromiseHandler_RequestPromise(t *testing.T) {
	bus := eventbus.New()
	mockFactory := &mockHermesCallerFactory{}
	aph := &HermesPromiseHandler{
		deps: HermesPromiseHandlerDeps{
			HermesURLGetter:      &mockHermesURLGetter{},
			HermesCallerFactory:  mockFactory.Get,
			Encryption:           &mockEncryptor{},
			EventBus:             eventbus.New(),
			HermesPromiseStorage: &mockHermesPromiseStorage{},
			FeeProvider:          &mockFeeProvider{},
		},
		queue: make(chan enqueuedRequest),
		stop:  make(chan struct{}),
	}
	aph.transactorFees = make(map[int64]registry.FeesResponse)
	err := aph.Subscribe(bus)
	assert.NoError(t, err)
	bus.Publish(event.AppTopicNode, event.Payload{
		Status: event.StatusStarted,
	})

	defer bus.Publish(event.AppTopicNode, event.Payload{
		Status: event.StatusStopped,
	})

	r := []byte{0x0, 0x1}
	em := crypto.ExchangeMessage{
		Promise: crypto.Promise{},
	}

	ch := aph.RequestPromise(r, em, identity.FromAddress("0x0000000000000000000000000000000000000001"), "session")

	err, more := <-ch
	assert.False(t, more)
	assert.Nil(t, err)
}

func TestHermesPromiseHandler_RequestPromise_BubblesErrors(t *testing.T) {
	bus := eventbus.New()
	mockFactory := &mockHermesCallerFactory{
		errToReturn: ErrNeedsRRecovery,
	}

	aph := &HermesPromiseHandler{
		deps: HermesPromiseHandlerDeps{
			HermesURLGetter:     &mockHermesURLGetter{},
			HermesCallerFactory: mockFactory.Get,
			Encryption: &mockEncryptor{
				errToReturn: errors.New("beep beep boop boop"),
			},
			EventBus:             bus,
			HermesPromiseStorage: &mockHermesPromiseStorage{},
			FeeProvider:          &mockFeeProvider{},
		},
		queue: make(chan enqueuedRequest),
		stop:  make(chan struct{}),
	}
	aph.transactorFees = make(map[int64]registry.FeesResponse)
	err := aph.Subscribe(bus)
	assert.NoError(t, err)
	bus.Publish(event.AppTopicNode, event.Payload{
		Status: event.StatusStarted,
	})

	defer bus.Publish(event.AppTopicNode, event.Payload{
		Status: event.StatusStopped,
	})

	r := []byte{0x0, 0x1}
	em := crypto.ExchangeMessage{
		Promise: crypto.Promise{},
	}

	ch := aph.RequestPromise(r, em, identity.FromAddress("0x0000000000000000000000000000000000000001"), "session")

	err, more := <-ch
	assert.True(t, more)
	assert.Error(t, err)

	err, more = <-ch
	assert.False(t, more)
	assert.Nil(t, err)
}

func TestHermesPromiseHandler_recoverR(t *testing.T) {
	type fields struct {
		deps       HermesPromiseHandlerDeps
		providerID identity.Identity
		chainID    int64
		hermesID   common.Address
	}
	mockFactory := &mockHermesCallerFactory{}
	tests := []struct {
		name    string
		fields  fields
		err     hermesError
		wantErr bool
		before  func()
	}{
		{
			name: "green path",
			fields: fields{
				deps: HermesPromiseHandlerDeps{
					HermesCallerFactory: mockFactory.Get,
					HermesURLGetter:     &mockHermesURLGetter{},
					Encryption:          &mockEncryptor{},
				},
				providerID: identity.FromAddress("0x0"),
			},
			err: HermesErrorResponse{
				ErrorMessage: `Secret R for previous promise exchange (Encrypted recovery data: "7b2272223a223731373736353731373736353731373736353731333133343333333433333334363137333634363636313733363636343733363436363738363337363332373336363634376136633733363136623637363136653632363136333632366436653631363436363663366236613631373336343636363137333636222c2261677265656d656e745f6964223a3132333435367d"`,
				CausedBy:     ErrNeedsRRecovery.Error(),
				c:            ErrNeedsRRecovery,
				ErrorData:    "7b2272223a223731373736353731373736353731373736353731333133343333333433333334363137333634363636313733363636343733363436363738363337363332373336363634376136633733363136623637363136653632363136333632366436653631363436363663366236613631373336343636363137333636222c2261677265656d656e745f6964223a3132333435367d",
			},
			wantErr: false,
			before: func() {
				mockFactory.errToReturn = nil
			},
		},
		{
			name: "bubbles hermes errors",
			fields: fields{
				providerID: identity.FromAddress("0x0"),
				deps: HermesPromiseHandlerDeps{
					HermesCallerFactory: mockFactory.Get,
					HermesURLGetter:     &mockHermesURLGetter{},
					Encryption:          &mockEncryptor{},
				},
			},
			err: HermesErrorResponse{
				ErrorMessage: `Secret R for previous promise exchange (Encrypted recovery data: "7b2272223a223731373736353731373736353731373736353731333133343333333433333334363137333634363636313733363636343733363436363738363337363332373336363634376136633733363136623637363136653632363136333632366436653631363436363663366236613631373336343636363137333636222c2261677265656d656e745f6964223a3132333435367d"`,
				CausedBy:     ErrNeedsRRecovery.Error(),
				c:            ErrNeedsRRecovery,
				ErrorData:    "7b2272223a223731373736353731373736353731373736353731333133343333333433333334363137333634363636313733363636343733363436363738363337363332373336363634376136633733363136623637363136653632363136333632366436653631363436363663366236613631373336343636363137333636222c2261677265656d656e745f6964223a3132333435367d",
			},
			wantErr: true,
			before: func() {
				mockFactory.errToReturn = errors.New("explosions")
			},
		},
		{
			name: "bubbles decryptor errors",
			fields: fields{
				providerID: identity.FromAddress("0x0"),
				deps: HermesPromiseHandlerDeps{
					HermesCallerFactory: mockFactory.Get,
					HermesURLGetter:     &mockHermesURLGetter{},
					Encryption: &mockEncryptor{
						errToReturn: errors.New("explosions"),
					},
				},
			},
			err: HermesErrorResponse{
				ErrorMessage: `Secret R for previous promise exchange (Encrypted recovery data: "7b2272223a223731373736353731373736353731373736353731333133343333333433333334363137333634363636313733363636343733363436363738363337363332373336363634376136633733363136623637363136653632363136333632366436653631363436363663366236613631373336343636363137333636222c2261677265656d656e745f6964223a3132333435367d"`,
				CausedBy:     ErrNeedsRRecovery.Error(),
				c:            ErrNeedsRRecovery,
				ErrorData:    "7b2272223a223731373736353731373736353731373736353731333133343333333433333334363137333634363636313733363636343733363436363738363337363332373336363634376136633733363136623637363136653632363136333632366436653631363436363663366236613631373336343636363137333636222c2261677265656d656e745f6964223a3132333435367d",
			},
			wantErr: true,
			before: func() {
				mockFactory.errToReturn = nil
			},
		},
	}
	for _, tt := range tests {
		if tt.before != nil {
			tt.before()
		}

		t.Run(tt.name, func(t *testing.T) {
			it := &HermesPromiseHandler{
				deps: tt.fields.deps,
			}
			if err := it.recoverR(tt.err, tt.fields.providerID, tt.fields.chainID, tt.fields.hermesID); (err != nil) != tt.wantErr {
				t.Errorf("HermesPromiseHandler.recoverR() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHermesPromiseHandler_handleHermesError(t *testing.T) {
	merr := errors.New("this is a test")
	mockFactory := &mockHermesCallerFactory{}
	tests := []struct {
		name       string
		err        error
		wantErr    error
		providerID identity.Identity
		hermesID   common.Address
		chainID    int64
		deps       HermesPromiseHandlerDeps
	}{
		{
			name:    "ignores nil errors",
			wantErr: nil,
			err:     nil,
		},
		{
			name:    "returns nil on ErrHermesNoPreviousPromise",
			wantErr: nil,
			err:     ErrHermesNoPreviousPromise,
		},
		{
			name:    "returns error if something else happens",
			wantErr: ErrTooManyRequests,
			err:     ErrTooManyRequests,
		},
		{
			name: "bubbles R recovery errors",
			deps: HermesPromiseHandlerDeps{
				HermesCallerFactory: mockFactory.Get,
				HermesURLGetter:     &mockHermesURLGetter{},
				Encryption: &mockEncryptor{
					errToReturn: merr,
				},
			},
			providerID: identity.FromAddress("0x0"),
			wantErr:    merr,
			err: HermesErrorResponse{
				c: ErrNeedsRRecovery,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			aph := &HermesPromiseHandler{
				deps: tt.deps,
			}
			err := aph.handleHermesError(tt.err, tt.providerID, tt.chainID, tt.hermesID)
			if tt.wantErr == nil {
				assert.NoError(t, err, tt.name)
			} else {
				log.Debug().Msgf("%v", err)
				assert.True(t, errors.Is(err, tt.wantErr), tt.name)
			}
		})
	}
}

type mockFeeProvider struct {
	toReturn    registry.FeesResponse
	errToReturn error
}

func (mfp *mockFeeProvider) FetchSettleFees(chainID int64) (registry.FeesResponse, error) {
	return mfp.toReturn, mfp.errToReturn
}

type mockHermesCallerFactory struct {
	errToReturn error
}

func (mhcf *mockHermesCallerFactory) Get(url string) HermesHTTPRequester {
	return &mockHermesCaller{
		errToReturn: mhcf.errToReturn,
	}
}

type mockHermesURLGetter struct {
	errToReturn error
	urlToReturn string
}

func (mhug *mockHermesURLGetter) GetHermesURL(chainID int64, address common.Address) (string, error) {
	return mhug.urlToReturn, mhug.errToReturn
}
