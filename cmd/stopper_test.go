/*
 * Copyright (C) 2017 The "MysteriumNetwork/node" Authors.
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

package cmd

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func newFakeKiller(killReturnsError bool) *fakeKiller {
	return &fakeKiller{
		Killed:           false,
		killReturnsError: killReturnsError,
	}
}

type fakeKiller struct {
	Killed           bool
	killReturnsError bool
}

func (fk *fakeKiller) Kill() error {
	fk.Killed = true
	if fk.killReturnsError {
		return errors.New("fake error")
	}
	return nil
}

func newFakeExiter() fakeExiter {
	return fakeExiter{-1}
}

type fakeExiter struct {
	LastCode int
}

func (fe *fakeExiter) Exit(code int) {
	fe.LastCode = code
}

func TestNewStopper(t *testing.T) {
	killer := newFakeKiller(false)
	exiter := newFakeExiter()
	stopper := newStopper(killer.Kill, exiter.Exit)
	assert.NotNil(t, stopper)
}

func TestStopperExitsWithSuccessWhenKillerSucceeds(t *testing.T) {
	exiter := newFakeExiter()
	fakeKiller := newFakeKiller(false)

	stopper := newStopper(fakeKiller.Kill, exiter.Exit)
	stopper()

	assert.Equal(t, 0, exiter.LastCode)
	assert.True(t, fakeKiller.Killed)
}

func TestStopperExitsWithErrorWhenKillerFails(t *testing.T) {
	exiter := newFakeExiter()
	fakeKiller := newFakeKiller(true)

	stopper := newStopper(fakeKiller.Kill, exiter.Exit)
	stopper()

	assert.Equal(t, 1, exiter.LastCode)
	assert.True(t, fakeKiller.Killed)
}
