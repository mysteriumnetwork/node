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
	"encoding/hex"
	"encoding/json"
	stdErr "errors"
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/mysteriumnetwork/node/core/node/event"
	"github.com/mysteriumnetwork/node/core/service/servicestate"
	"github.com/mysteriumnetwork/node/eventbus"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/identity/registry"
	sessionEvent "github.com/mysteriumnetwork/node/session/event"
	pinge "github.com/mysteriumnetwork/node/session/pingpong/event"
	"github.com/mysteriumnetwork/payments/crypto"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

type accountantPromiseStorage interface {
	Store(providerID identity.Identity, accountantID common.Address, promise AccountantPromise) error
	Get(providerID identity.Identity, accountantID common.Address) (AccountantPromise, error)
}

type feeProvider interface {
	FetchSettleFees() (registry.FeesResponse, error)
}

type accountantCaller interface {
	RequestPromise(rp RequestPromise) (crypto.Promise, error)
	RevealR(r string, provider string, agreementID uint64) error
}

type encryption interface {
	Decrypt(addr common.Address, encrypted []byte) ([]byte, error)
	Encrypt(addr common.Address, plaintext []byte) ([]byte, error)
}

// AccountantPromiseHandlerDeps represents the AccountantPromiseHandler dependencies.
type AccountantPromiseHandlerDeps struct {
	AccountantPromiseStorage accountantPromiseStorage
	AccountantCaller         accountantCaller
	AccountantID             common.Address
	FeeProvider              feeProvider
	Encryption               encryption
	EventBus                 eventbus.Publisher
}

// AccountantPromiseHandler handles the accountant promises for ongoing sessions.
type AccountantPromiseHandler struct {
	deps          AccountantPromiseHandlerDeps
	queue         chan enqueuedRequest
	stop          chan struct{}
	stopOnce      sync.Once
	startOnce     sync.Once
	transactorFee registry.FeesResponse
}

// NewAccountantPromiseHandler returns a new instance of accountant promise handler.
func NewAccountantPromiseHandler(deps AccountantPromiseHandlerDeps) *AccountantPromiseHandler {
	return &AccountantPromiseHandler{
		deps:  deps,
		queue: make(chan enqueuedRequest, 100),
		stop:  make(chan struct{}),
	}
}

type enqueuedRequest struct {
	errChan    chan error
	r          []byte
	em         crypto.ExchangeMessage
	providerID identity.Identity
	sessionID  string
}

// RequestPromise adds the request to the queue.
func (aph *AccountantPromiseHandler) RequestPromise(r []byte, em crypto.ExchangeMessage, providerID identity.Identity, sessionID string) <-chan error {
	er := enqueuedRequest{
		r:          r,
		em:         em,
		providerID: providerID,
		errChan:    make(chan error),
		sessionID:  sessionID,
	}
	aph.queue <- er
	return er.errChan
}

func (aph *AccountantPromiseHandler) updateFee() {
	fees, err := aph.deps.FeeProvider.FetchSettleFees()
	if err != nil {
		log.Warn().Err(err).Msg("could not fetch fees, ignoring")
		return
	}

	aph.transactorFee = fees
}

func (aph *AccountantPromiseHandler) handleRequests() {
	log.Debug().Msgf("accountant promise handler started")
	defer log.Debug().Msgf("accountant promise handler stopped")
	for {
		select {
		case <-aph.stop:
			return
		case entry := <-aph.queue:
			aph.requestPromise(entry)
		}
	}
}

// Subscribe subscribes AccountantPromiseHandler to relevant events.
func (aph *AccountantPromiseHandler) Subscribe(bus eventbus.Subscriber) error {
	err := bus.SubscribeAsync(event.AppTopicNode, aph.handleNodeStopEvents)
	if err != nil {
		return fmt.Errorf("could not subscribe to node events: %w", err)
	}

	err = bus.SubscribeAsync(servicestate.AppTopicServiceStatus, aph.handleServiceEvent)
	if err != nil {
		return fmt.Errorf("could not subscribe to service events: %w", err)
	}
	return nil
}

func (aph *AccountantPromiseHandler) handleServiceEvent(ev servicestate.AppEventServiceStatus) {
	if ev.Status == string(servicestate.Running) {
		aph.startOnce.Do(
			func() {
				aph.updateFee()
				aph.handleRequests()
			})
	}
}

func (aph *AccountantPromiseHandler) doStop() {
	aph.stopOnce.Do(func() {
		close(aph.stop)
	})
}

func (aph *AccountantPromiseHandler) handleNodeStopEvents(e event.Payload) {
	if e.Status == event.StatusStopped {
		aph.doStop()
		return
	}
}

func (aph *AccountantPromiseHandler) requestPromise(er enqueuedRequest) {
	defer func() { close(er.errChan) }()

	if !aph.transactorFee.IsValid() {
		aph.updateFee()
	}

	details := rRecoveryDetails{
		R:           hex.EncodeToString(er.r),
		AgreementID: er.em.AgreementID,
	}

	bytes, err := json.Marshal(details)
	if err != nil {
		er.errChan <- fmt.Errorf("could not marshal R recovery details: %w", err)
		return
	}

	encrypted, err := aph.deps.Encryption.Encrypt(er.providerID.ToCommonAddress(), bytes)
	if err != nil {
		er.errChan <- fmt.Errorf("could not encrypt R: %w", err)
		return
	}

	request := RequestPromise{
		ExchangeMessage: er.em,
		TransactorFee:   aph.transactorFee.Fee,
		RRecoveryData:   hex.EncodeToString(encrypted),
	}

	promise, err := aph.deps.AccountantCaller.RequestPromise(request)
	err = aph.handleAccountantError(err, er.providerID)
	if err != nil {
		er.errChan <- fmt.Errorf("accountant request promise error: %w", err)
		return
	}

	ap := AccountantPromise{
		Promise:     promise,
		R:           hex.EncodeToString(er.r),
		Revealed:    false,
		AgreementID: er.em.AgreementID,
	}

	err = aph.deps.AccountantPromiseStorage.Store(er.providerID, aph.deps.AccountantID, ap)
	if err != nil && !stdErr.Is(err, ErrAttemptToOverwrite) {
		er.errChan <- fmt.Errorf("could not store accountant promise: %w", err)
		return
	}

	aph.deps.EventBus.Publish(pinge.AppTopicAccountantPromise, pinge.AppEventAccountantPromise{
		Promise:      promise,
		AccountantID: aph.deps.AccountantID,
		ProviderID:   er.providerID,
	})
	aph.deps.EventBus.Publish(sessionEvent.AppTopicSessionTokensEarned, sessionEvent.AppEventSessionTokensEarned{
		ProviderID: er.providerID,
		SessionID:  er.sessionID,
		Total:      er.em.AgreementTotal,
	})

	err = aph.revealR(er.providerID)
	err = aph.handleAccountantError(err, er.providerID)
	if err != nil {
		er.errChan <- fmt.Errorf("accountant reveal r error: %w", err)
		return
	}
}

func (aph *AccountantPromiseHandler) revealR(providerID identity.Identity) error {
	needsRevealing := false
	accountantPromise, err := aph.deps.AccountantPromiseStorage.Get(providerID, aph.deps.AccountantID)
	switch err {
	case nil:
		needsRevealing = !accountantPromise.Revealed
	case ErrNotFound:
		needsRevealing = false
	default:
		return fmt.Errorf("could not get accountant promise: %w", err)
	}

	if !needsRevealing {
		return nil
	}

	err = aph.deps.AccountantCaller.RevealR(accountantPromise.R, providerID.Address, accountantPromise.AgreementID)
	handledErr := aph.handleAccountantError(err, providerID)
	if handledErr != nil {
		return fmt.Errorf("could not reveal R: %w", err)
	}

	accountantPromise.Revealed = true
	err = aph.deps.AccountantPromiseStorage.Store(providerID, aph.deps.AccountantID, accountantPromise)
	if err != nil && !stdErr.Is(err, ErrAttemptToOverwrite) {
		return fmt.Errorf("could not store accountant promise: %w", err)
	}

	return nil
}

func (aph *AccountantPromiseHandler) handleAccountantError(err error, providerID identity.Identity) error {
	if err == nil {
		return nil
	}

	switch {
	case stdErr.Is(err, ErrNeedsRRecovery):
		var aer AccountantErrorResponse
		ok := stdErr.As(err, &aer)
		if !ok {
			return errors.New("could not cast errNeedsRecovery to accountantError")
		}
		recoveryErr := aph.recoverR(aer, providerID)
		if recoveryErr != nil {
			return recoveryErr
		}
		return nil
	case stdErr.Is(err, ErrAccountantNoPreviousPromise):
		log.Info().Msg("no previous promise on accountant, will mark R as revealed")
		return nil
	default:
		return err
	}
}

func (aph *AccountantPromiseHandler) recoverR(aerr accountantError, providerID identity.Identity) error {
	log.Info().Msg("Recovering R...")
	decoded, err := hex.DecodeString(aerr.Data())
	if err != nil {
		return fmt.Errorf("could not decode R recovery details: %w", err)
	}

	decrypted, err := aph.deps.Encryption.Decrypt(providerID.ToCommonAddress(), decoded)
	if err != nil {
		return fmt.Errorf("could not decrypt R details: %w", err)
	}

	res := rRecoveryDetails{}
	err = json.Unmarshal(decrypted, &res)
	if err != nil {
		return fmt.Errorf("could not unmarshal R details: %w", err)
	}

	log.Info().Msg("R recovered, will reveal...")
	err = aph.deps.AccountantCaller.RevealR(res.R, providerID.Address, res.AgreementID)
	if err != nil {
		return fmt.Errorf("could not reveal R: %w", err)
	}

	log.Info().Msg("R recovered successfully")
	return nil
}
