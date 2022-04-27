/*
 * Copyright (C) 2021 The "MysteriumNetwork/node" Authors.
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
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/rs/zerolog/log"
)

// HermesStatusChecker checks hermes activity and caches the results.
type HermesStatusChecker struct {
	mbc           mbc
	observer      observerApi
	cacheDuration time.Duration

	cachedValues map[string]HermesStatus
	lock         sync.Mutex
}

// NewHermesStatusChecker creates a new instance of hermes activity checker.
func NewHermesStatusChecker(bc mbc, observer observerApi, cacheDuration time.Duration) *HermesStatusChecker {
	return &HermesStatusChecker{
		cachedValues:  make(map[string]HermesStatus),
		cacheDuration: cacheDuration,
		mbc:           bc,
		observer:      observer,
	}
}

// HermesStatus represents the hermes status.
type HermesStatus struct {
	HermesID   common.Address
	ChainID    int64
	IsActive   bool
	Fee        uint16
	ValidUntil time.Time
}

func (has HermesStatus) isValid() bool {
	return time.Now().Before(has.ValidUntil)
}

type mbc interface {
	IsHermesActive(chainID int64, hermesID common.Address) (bool, error)
	IsHermesRegistered(chainID int64, registryAddress, hermesID common.Address) (bool, error)
	GetHermesFee(chainID int64, hermesAddress common.Address) (uint16, error)
}

// GetHermesStatus determines if hermes is active or not.
func (hac *HermesStatusChecker) GetHermesStatus(chainID int64, registryAddress common.Address, hermesID common.Address) (HermesStatus, error) {
	cached, ok := hac.getFromCache(chainID, hermesID)
	if ok {
		return cached, nil
	}

	status, err := hac.fetchHermesStatus(chainID, registryAddress, hermesID)
	if err != nil {
		return HermesStatus{}, err
	}

	status = hac.setInCache(chainID, hermesID, status.IsActive, status.Fee)
	return status, nil
}

func (hac *HermesStatusChecker) formKey(chainID int64, hermesID common.Address) string {
	return fmt.Sprintf("%v_%v", hermesID.Hex(), chainID)
}

func (hac *HermesStatusChecker) fetchHermesStatus(chainID int64, registryAddress common.Address, hermesID common.Address) (HermesStatus, error) {
	// hermes is active if: is registered and is active.
	isRegistered, err := hac.mbc.IsHermesRegistered(chainID, registryAddress, hermesID)
	if err != nil {
		log.Err(err).Msg("using observer as fallback")
		return hac.fetchFallbackHermesStatus(chainID, hermesID)
	}

	isActive, err := hac.mbc.IsHermesActive(chainID, hermesID)
	if err != nil {
		return HermesStatus{}, fmt.Errorf("could not check if hermes(%v) is active on chain %v: %w", hermesID.Hex(), chainID, err)
	}

	fee, err := hac.mbc.GetHermesFee(chainID, hermesID)
	if err != nil {
		return HermesStatus{}, fmt.Errorf("could not check hermes(%v) fee on chain %v: %w", hermesID.Hex(), chainID, err)
	}

	status := HermesStatus{
		Fee:      fee,
		IsActive: isRegistered && isActive,
	}

	return status, nil
}

func (hac *HermesStatusChecker) fetchFallbackHermesStatus(chainID int64, hermesID common.Address) (HermesStatus, error) {
	hermesData, err := hac.observer.GetHermesData(chainID, hermesID)
	if err != nil {
		return HermesStatus{}, fmt.Errorf("failed to get hermes data from observer: %w", err)
	}
	return HermesStatus{
		IsActive: hermesData.Approved,
		Fee:      uint16(hermesData.Fee),
	}, nil

}

func (hac *HermesStatusChecker) setInCache(chainID int64, hermesID common.Address, isActive bool, fee uint16) HermesStatus {
	hac.lock.Lock()
	defer hac.lock.Unlock()
	status := HermesStatus{
		HermesID:   hermesID,
		IsActive:   isActive,
		ChainID:    chainID,
		ValidUntil: time.Now().Add(hac.cacheDuration),
		Fee:        fee,
	}
	hac.cachedValues[hac.formKey(chainID, hermesID)] = status
	return status
}

func (hac *HermesStatusChecker) getFromCache(chainID int64, hermesID common.Address) (HermesStatus, bool) {
	hac.lock.Lock()
	defer hac.lock.Unlock()

	v, ok := hac.cachedValues[hac.formKey(chainID, hermesID)]
	if !ok {
		return HermesStatus{}, false
	}

	if v.isValid() {
		return v, true
	}

	return HermesStatus{}, false
}
