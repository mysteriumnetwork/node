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

package actionstack

import (
	"errors"
	"sync"
)

var (
	// ErrAlreadyRun error is throwed with panic on Push or Run invocation
	// after Run was invoked at least once
	ErrAlreadyRun = errors.New("actions already fired")
)

// Action represents stackable action.
type Action func()

// ActionStack is a stack of actions which are executed in reverse order
// they were added
type ActionStack struct {
	stack []Action
	fired bool
	mu    sync.Mutex
}

// NewActionStack creates empty ActionStack
func NewActionStack() *ActionStack {
	return new(ActionStack)
}

// Run executes pushed actions in reverse order (FILO)
func (a *ActionStack) Run() {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.fired {
		panic(ErrAlreadyRun)
	}

	for i := len(a.stack) - 1; i >= 0; i-- {
		a.stack[i]()
	}
	a.stack = nil
	a.fired = true
}

// Push adds new action on top of stack. Last added action will be executed
// by Run() first.
func (a *ActionStack) Push(elems ...Action) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.fired {
		panic(ErrAlreadyRun)
	}

	a.stack = append(a.stack, elems...)
}
