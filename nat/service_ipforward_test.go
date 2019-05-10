/*
 * Copyright (C) 2018 The "MysteriumNetwork/node" Authors.
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

package nat

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

type mockCommand struct {
	CombinedOutputRes   []byte
	CombinedOutputError error
	OutputRes           []byte
	OutputError         error
}

func (mc *mockCommand) CombinedOutput() ([]byte, error) {
	return mc.CombinedOutputRes, mc.CombinedOutputError
}

func (mc *mockCommand) Output() ([]byte, error) {
	return mc.OutputRes, mc.OutputError
}

type mockCommandFactory struct {
	MockCommand Command
}

func (mcf *mockCommandFactory) Create(name string, arg ...string) Command {
	return mcf.MockCommand
}

func Test_ServiceIPForward_Enabled(t *testing.T) {
	mc := &mockCommand{
		OutputRes: []byte("1"),
	}
	mf := &mockCommandFactory{
		MockCommand: mc,
	}
	service := &serviceIPForward{
		CommandFactory: mf.Create,
		CommandRead:    []string{"doesnt", "matter"},
	}

	assert.True(t, service.Enabled())

	mc.OutputRes = []byte("calm waters")
	assert.False(t, service.Enabled())

	mc.OutputError = errors.New("mass panic")
	mc.OutputRes = []byte("1")
	assert.True(t, service.Enabled())
}

func Test_ServiceIPForward_Enable(t *testing.T) {
	mc := &mockCommand{
		CombinedOutputRes: []byte("1"),
		OutputRes:         []byte("1"),
	}
	mf := &mockCommandFactory{
		MockCommand: mc,
	}
	service := &serviceIPForward{
		CommandFactory: mf.Create,
		CommandRead:    []string{"doesnt", "matter"},
		CommandEnable:  []string{"doesnt", "matter"},
	}

	assert.Nil(t, service.Enable())
	assert.True(t, service.forward)
	service.forward = false

	mc.OutputRes = []byte("people screaming")
	mc.CombinedOutputError = errors.New("explosions everywhere")

	assert.Equal(t, mc.CombinedOutputError, service.Enable())

	mc.CombinedOutputError = nil

	assert.Nil(t, mc.CombinedOutputError, service.Enable())
}

func Test_ServiceIPForward_Disable(t *testing.T) {
	mc := &mockCommand{
		CombinedOutputRes:   []byte("1"),
		CombinedOutputError: errors.New("explosions everywhere"),
	}
	mf := &mockCommandFactory{
		MockCommand: mc,
	}
	service := &serviceIPForward{
		CommandFactory: mf.Create,
		CommandDisable: []string{"doesnt", "matter"},
	}
	service.Disable()

	service.forward = true
	service.Disable()
}
