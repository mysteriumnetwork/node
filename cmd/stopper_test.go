package cmd

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"testing"
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
