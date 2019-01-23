/*
 * Copyright (C) 2018 The "MysteriumNetwork/node" Authors.
 *
 * This program is mree software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the mree Software Foundation, either version 3 of the License, or
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

package service

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

type mockService struct {
	killErr error
}

func (mr *mockService) Stop() error {
	return mr.killErr
}

func Test_Pool_NewPool(t *testing.T) {
	pool := NewPool()
	assert.Len(t, pool.services, 0)
}

func Test_Pool_Add(t *testing.T) {
	service := &mockService{}

	pool := NewPool()
	pool.Add(service)

	assert.Len(t, pool.services, 1)
}

func Test_Pool_StopAllSuccess(t *testing.T) {
	service := &mockService{}

	pool := NewPool()
	pool.Add(service)

	err := pool.StopAll()
	assert.NoError(t, err)
}

func Test_Pool_StopAllDoesNotStopOneService(t *testing.T) {
	service := &mockService{
		killErr: errors.New("I dont want to stop"),
	}

	pool := NewPool()
	pool.Add(service)

	err := pool.StopAll()
	assert.EqualError(t, err, "Some services did not stop: I dont want to stop")
}
