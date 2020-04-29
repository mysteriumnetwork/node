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

	"github.com/mysteriumnetwork/node/core/node/event"
	"github.com/mysteriumnetwork/node/core/service/servicestate"
	"github.com/mysteriumnetwork/node/eventbus"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/identity/registry"
	"github.com/mysteriumnetwork/payments/crypto"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
)

func TestAccountantPromiseHandler_RequestPromise(t *testing.T) {
	bus := eventbus.New()
	aph := &AccountantPromiseHandler{
		deps: AccountantPromiseHandlerDeps{
			AccountantCallerFactory:  func(string) AccCaller { return &mockAccountantCaller{} },
			Encryption:               &mockEncryptor{},
			EventBus:                 eventbus.New(),
			AccountantPromiseStorage: &mockAccountantPromiseStorage{},
			FeeProvider:              &mockFeeProvider{},
		},
		queue: make(chan enqueuedRequest),
		stop:  make(chan struct{}),
	}
	err := aph.Subscribe(bus)
	assert.NoError(t, err)
	bus.Publish(servicestate.AppTopicServiceStatus, servicestate.AppEventServiceStatus{
		Status: string(servicestate.Running),
	})
	defer bus.Publish(event.AppTopicNode, event.Payload{
		Status: event.StatusStopped,
	})

	r := []byte{0x0, 0x1}
	em := crypto.ExchangeMessage{
		Promise: crypto.Promise{},
	}

	ch := aph.RequestPromise(r, em, identity.FromAddress("asddadadqweqwe"), "session", "")

	err, more := <-ch
	assert.False(t, more)
	assert.Nil(t, err)
}

func TestAccountantPromiseHandler_RequestPromise_BubblesErrors(t *testing.T) {
	bus := eventbus.New()
	aph := &AccountantPromiseHandler{
		deps: AccountantPromiseHandlerDeps{
			AccountantCallerFactory: func(string) AccCaller {
				return &mockAccountantCaller{
					errToReturn: ErrNeedsRRecovery,
				}
			},
			Encryption: &mockEncryptor{
				errToReturn: errors.New("beep beep boop boop"),
			},
			EventBus:                 bus,
			AccountantPromiseStorage: &mockAccountantPromiseStorage{},
			FeeProvider:              &mockFeeProvider{},
		},
		queue: make(chan enqueuedRequest),
		stop:  make(chan struct{}),
	}
	err := aph.Subscribe(bus)
	assert.NoError(t, err)
	bus.Publish(servicestate.AppTopicServiceStatus, servicestate.AppEventServiceStatus{
		Status: string(servicestate.Running),
	})
	defer bus.Publish(event.AppTopicNode, event.Payload{
		Status: event.StatusStopped,
	})

	r := []byte{0x0, 0x1}
	em := crypto.ExchangeMessage{
		Promise: crypto.Promise{},
	}

	ch := aph.RequestPromise(r, em, identity.FromAddress("asddadadqweqwe"), "session", "")

	err, more := <-ch
	assert.True(t, more)
	assert.Error(t, err)

	err, more = <-ch
	assert.False(t, more)
	assert.Nil(t, err)
}

func TestAccountantPromiseHandler_recoverR(t *testing.T) {
	type fields struct {
		deps       AccountantPromiseHandlerDeps
		providerID identity.Identity
	}
	tests := []struct {
		name    string
		fields  fields
		err     accountantError
		wantErr bool
	}{
		{
			name: "green path",
			fields: fields{
				deps: AccountantPromiseHandlerDeps{
					AccountantCallerFactory: func(string) AccCaller {
						return &mockAccountantCaller{}
					},
					Encryption: &mockEncryptor{},
				},
				providerID: identity.FromAddress("0x0"),
			},
			err: AccountantErrorResponse{
				ErrorMessage: `Secret R for previous promise exchange (Encrypted recovery data: "7b2272223a223731373736353731373736353731373736353731333133343333333433333334363137333634363636313733363636343733363436363738363337363332373336363634376136633733363136623637363136653632363136333632366436653631363436363663366236613631373336343636363137333636222c2261677265656d656e745f6964223a3132333435367d"`,
				CausedBy:     ErrNeedsRRecovery.Error(),
				c:            ErrNeedsRRecovery,
				ErrorData:    "7b2272223a223731373736353731373736353731373736353731333133343333333433333334363137333634363636313733363636343733363436363738363337363332373336363634376136633733363136623637363136653632363136333632366436653631363436363663366236613631373336343636363137333636222c2261677265656d656e745f6964223a3132333435367d",
			},
			wantErr: false,
		},
		{
			name: "bubbles accountant errors",
			fields: fields{
				providerID: identity.FromAddress("0x0"),
				deps: AccountantPromiseHandlerDeps{
					AccountantCallerFactory: func(string) AccCaller {
						return &mockAccountantCaller{
							errToReturn: errors.New("explosions"),
						}
					},
					Encryption: &mockEncryptor{},
				},
			},
			err: AccountantErrorResponse{
				ErrorMessage: `Secret R for previous promise exchange (Encrypted recovery data: "7b2272223a223731373736353731373736353731373736353731333133343333333433333334363137333634363636313733363636343733363436363738363337363332373336363634376136633733363136623637363136653632363136333632366436653631363436363663366236613631373336343636363137333636222c2261677265656d656e745f6964223a3132333435367d"`,
				CausedBy:     ErrNeedsRRecovery.Error(),
				c:            ErrNeedsRRecovery,
				ErrorData:    "7b2272223a223731373736353731373736353731373736353731333133343333333433333334363137333634363636313733363636343733363436363738363337363332373336363634376136633733363136623637363136653632363136333632366436653631363436363663366236613631373336343636363137333636222c2261677265656d656e745f6964223a3132333435367d",
			},
			wantErr: true,
		},
		{
			name: "bubbles decryptor errors",
			fields: fields{
				providerID: identity.FromAddress("0x0"),
				deps: AccountantPromiseHandlerDeps{
					AccountantCallerFactory: func(string) AccCaller {
						return &mockAccountantCaller{}
					},
					Encryption: &mockEncryptor{
						errToReturn: errors.New("explosions"),
					},
				},
			},
			err: AccountantErrorResponse{
				ErrorMessage: `Secret R for previous promise exchange (Encrypted recovery data: "7b2272223a223731373736353731373736353731373736353731333133343333333433333334363137333634363636313733363636343733363436363738363337363332373336363634376136633733363136623637363136653632363136333632366436653631363436363663366236613631373336343636363137333636222c2261677265656d656e745f6964223a3132333435367d"`,
				CausedBy:     ErrNeedsRRecovery.Error(),
				c:            ErrNeedsRRecovery,
				ErrorData:    "7b2272223a223731373736353731373736353731373736353731333133343333333433333334363137333634363636313733363636343733363436363738363337363332373336363634376136633733363136623637363136653632363136333632366436653631363436363663366236613631373336343636363137333636222c2261677265656d656e745f6964223a3132333435367d",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			it := &AccountantPromiseHandler{
				deps: tt.fields.deps,
			}
			if err := it.recoverR(tt.err, enqueuedRequest{
				providerID: tt.fields.providerID,
			}); (err != nil) != tt.wantErr {
				t.Errorf("AccountantPromiseHandler.recoverR() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAccountantPromiseHandler_handleAccountantError(t *testing.T) {
	merr := errors.New("this is a test")
	tests := []struct {
		name       string
		err        error
		wantErr    error
		providerID identity.Identity
		deps       AccountantPromiseHandlerDeps
	}{
		{
			name:    "ignores nil errors",
			wantErr: nil,
			err:     nil,
		},
		{
			name:    "returns nil on ErrAccountantNoPreviousPromise",
			wantErr: nil,
			err:     ErrAccountantNoPreviousPromise,
		},
		{
			name:    "returns error if something else happens",
			wantErr: ErrTooManyRequests,
			err:     ErrTooManyRequests,
		},
		{
			name: "bubbles R recovery errors",
			deps: AccountantPromiseHandlerDeps{
				AccountantCallerFactory: func(string) AccCaller {
					return &mockAccountantCaller{}
				},
				Encryption: &mockEncryptor{
					errToReturn: merr,
				},
			},
			providerID: identity.FromAddress("0x0"),
			wantErr:    merr,
			err: AccountantErrorResponse{
				c: ErrNeedsRRecovery,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			aph := &AccountantPromiseHandler{
				deps: tt.deps,
			}
			err := aph.handleAccountantError(tt.err, enqueuedRequest{
				providerID: tt.providerID,
			})
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

func (mfp *mockFeeProvider) FetchSettleFees() (registry.FeesResponse, error) {
	return mfp.toReturn, mfp.errToReturn
}
