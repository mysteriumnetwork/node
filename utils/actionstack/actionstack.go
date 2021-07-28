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

package actionstack

// Action represents stackable action.
type Action func()

// ActionStack is a stack of actions which are executed in reverse order
// they were added
type ActionStack struct {
	stack []Action
}

// NewActionStack creates empty ActionStack
func NewActionStack() *ActionStack {
	return new(ActionStack)
}

// Run executes pushed actions in reverse order (FILO)
func (a *ActionStack) Run() {
	for i := len(a.stack) - 1; i >= 0; i-- {
		a.stack[i]()
	}
}

// Push adds new action on top of stack. Last added action will be executed
// by Run() first.
func (a *ActionStack) Push(elems ...Action) {
	a.stack = append(a.stack, elems...)
}
