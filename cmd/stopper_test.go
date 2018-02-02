package cmd

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"testing"
)

func NewFakeKiller(killReturnsError bool) *fakeKiller {
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

func NewFakeExiter() fakeExiter {
	return fakeExiter{-1}
}

type fakeExiter struct {
	LastCode int
}

func (fe *fakeExiter) Exit(code int) {
	fe.LastCode = code
}

func TestNewStopper(t *testing.T) {
	killer := NewFakeKiller(false)
	exiter := NewFakeExiter()
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
			[]*fakeKiller{NewFakeKiller(false)},
			0,
		},
		// Failing killer
		{
			[]*fakeKiller{NewFakeKiller(true)},
			1,
		},
		// Two successful killers
		{
			[]*fakeKiller{NewFakeKiller(false), NewFakeKiller(false)},
			0,
		},
		// First killer fails, second gets executed
		{
			[]*fakeKiller{NewFakeKiller(true), NewFakeKiller(false)},
			1,
		},
	}

	for _, test := range tests {
		exiter := NewFakeExiter()
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
