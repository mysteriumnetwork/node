/*
 * Copyright (C) 2019 The "MysteriumNetwork/node" Authors.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the free Software Foundation, either version 3 of the License, or
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

type mockPublisher struct {
	publishedTopic string
	publishedData  interface{}
}

func (mockPublisher *mockPublisher) Publish(topic string, data interface{}) {
	mockPublisher.publishedTopic = topic
	mockPublisher.publishedData = data
}

func (mr *mockService) Stop() error {
	return mr.killErr
}

func Test_Pool_NewPool(t *testing.T) {
	pool := NewPool(&mockPublisher{})
	assert.Len(t, pool.instances, 0)
}

func Test_Pool_Add(t *testing.T) {
	instance := &Instance{}

	pool := NewPool(&mockPublisher{})
	pool.Add(instance)

	assert.Len(t, pool.instances, 1)
}

func Test_Pool_StopAllSuccess(t *testing.T) {
	instance := &Instance{
		service: &mockService{},
	}

	pool := NewPool(&mockPublisher{})
	pool.Add(instance)

	err := pool.StopAll()
	assert.NoError(t, err)
}

func Test_Pool_StopDoesNotStop(t *testing.T) {
	service := &mockService{killErr: errors.New("I dont want to stop")}
	instance := &Instance{id: "test id", service: service}

	pool := NewPool(&mockPublisher{})
	pool.Add(instance)

	err := pool.Stop("test id")
	assert.EqualError(t, err, "ErrorCollection(I dont want to stop)")
}

func Test_Pool_StopReturnsErrIfInstanceDoesNotExist(t *testing.T) {
	pool := NewPool(&mockPublisher{})
	err := pool.Stop("something")
	assert.Equal(t, ErrNoSuchInstance, err)
}

func Test_Pool_StopAllDoesNotStopOneInstance(t *testing.T) {
	service := &mockService{killErr: errors.New("I dont want to stop")}
	instance := &Instance{id: "test id", service: service}

	pool := NewPool(&mockPublisher{})
	pool.Add(instance)

	err := pool.StopAll()
	assert.EqualError(t, err, "Some instances did not stop: ErrorCollection(I dont want to stop)")
}
