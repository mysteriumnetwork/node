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
	stopper := newStopper(exiter.Exit, killer.Kill)
	assert.NotNil(t, stopper)
}

func TestStop(t *testing.T) {
	tests := []struct {
		fakeKillers      []*fakeKiller
		expectedExitCode int
	}{
		// Successful killer
		{
			[]*fakeKiller{newFakeKiller(false)},
			0,
		},
		// Failing killer
		{
			[]*fakeKiller{newFakeKiller(true)},
			1,
		},
		// Two successful killers
		{
			[]*fakeKiller{newFakeKiller(false), newFakeKiller(false)},
			0,
		},
		// First killer fails, second gets executed
		{
			[]*fakeKiller{newFakeKiller(true), newFakeKiller(false)},
			1,
		},
	}

	for _, test := range tests {
		exiter := newFakeExiter()
		var killers []Killer
		for _, fakeKiller := range test.fakeKillers {
			killers = append(killers, fakeKiller.Kill)
		}

		stopper := newStopper(exiter.Exit, killers...)
		stopper()

		assert.Equal(t, test.expectedExitCode, exiter.LastCode)
		for _, fakeKiller := range test.fakeKillers {
			assert.True(t, fakeKiller.Killed)
		}
	}
}
