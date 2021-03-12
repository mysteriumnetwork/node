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
)

// HermesActivityChecker checks hermes activity and caches the results.
type HermesActivityChecker struct {
	mbc           mbc
	cacheDuration time.Duration

	cachedValues map[string]hermesActivityStatus
	lock         sync.Mutex
}

// NewHermesActivityChecker creates a new instance of hermes activity checker.
func NewHermesActivityChecker(bc mbc, cacheDuration time.Duration) *HermesActivityChecker {
	return &HermesActivityChecker{
		cachedValues:  make(map[string]hermesActivityStatus),
		cacheDuration: cacheDuration,
		mbc:           bc,
	}
}

type hermesActivityStatus struct {
	hermesID   common.Address
	chainID    int64
	isActive   bool
	validUntil time.Time
}

func (has hermesActivityStatus) isValid() bool {
	return time.Now().Before(has.validUntil)
}

type mbc interface {
	IsHermesActive(chainID int64, hermesID common.Address) (bool, error)
	IsHermesRegistered(chainID int64, registryAddress, hermesID common.Address) (bool, error)
}

// IsHermesActive determines if hermes is active or not.
func (hac *HermesActivityChecker) IsHermesActive(chainID int64, registryAddress common.Address, hermesID common.Address) (bool, error) {
	cached, ok := hac.getFromCache(chainID, hermesID)
	if ok {
		return cached.isActive, nil
	}

	active, err := hac.isHermesActive(chainID, registryAddress, hermesID)
	if err != nil {
		return false, err
	}

	hac.setInCache(chainID, hermesID, active)
	return active, nil
}

func (hac *HermesActivityChecker) formKey(chainID int64, hermesID common.Address) string {
	return fmt.Sprintf("%v_%v", hermesID.Hex(), chainID)
}

func (hac *HermesActivityChecker) isHermesActive(chainID int64, registryAddress common.Address, hermesID common.Address) (bool, error) {
	// hermes is active if: is registered and is active.
	isRegistered, err := hac.mbc.IsHermesRegistered(chainID, registryAddress, hermesID)
	if err != nil {
		return false, fmt.Errorf("could not check if hermes(%v) is registered on chain %v via registry(%v): %w", hermesID.Hex(), chainID, registryAddress.Hex(), err)
	}

	isActive, err := hac.mbc.IsHermesActive(chainID, hermesID)
	if err != nil {
		return false, fmt.Errorf("could not check if hermes(%v) is active on chain %v: %w", hermesID.Hex(), chainID, err)
	}

	return isRegistered && isActive, nil
}

func (hac *HermesActivityChecker) setInCache(chainID int64, hermesID common.Address, isActive bool) {
	hac.lock.Lock()
	defer hac.lock.Unlock()

	hac.cachedValues[hac.formKey(chainID, hermesID)] = hermesActivityStatus{
		hermesID:   hermesID,
		isActive:   isActive,
		chainID:    chainID,
		validUntil: time.Now().Add(hac.cacheDuration),
	}
}

func (hac *HermesActivityChecker) getFromCache(chainID int64, hermesID common.Address) (hermesActivityStatus, bool) {
	hac.lock.Lock()
	defer hac.lock.Unlock()

	v, ok := hac.cachedValues[hac.formKey(chainID, hermesID)]
	if !ok {
		return hermesActivityStatus{}, false
	}

	if v.isValid() {
		return v, true
	}

	return hermesActivityStatus{}, false
}
