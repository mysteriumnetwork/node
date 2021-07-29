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
	"testing"
)

func TestMain(t *testing.T) {
	as := NewActionStack()
	var arr []int
	as.Push(func() {
		arr = append(arr, 1)
	})
	as.Push(
		func() {
			arr = append(arr, 2)
		},
		func() {
			arr = append(arr, 3)
		},
	)

	as.Run()

	if len(arr) != 3 {
		t.Fail()
	}
	for i, v := range []int{3, 2, 1} {
		if arr[i] != v {
			t.Fail()
		}
	}
}

func TestEmpty(t *testing.T) {
	// Should at least not panic due to some nil-values
	NewActionStack().Run()
}

func TestRunPanicsAfterRun(t *testing.T) {
	var failedSuccessfully bool
	as := NewActionStack()
	as.Push(func() {})
	as.Run()

	func() {
		defer func() {
			if r := recover(); r != nil {
				failedSuccessfully = true
			}
		}()
		as.Run()
	}()

	if !failedSuccessfully {
		t.Fail()
	}
}

func TestAddPanicsAfterRun(t *testing.T) {
	var failedSuccessfully bool
	as := NewActionStack()
	as.Run()

	func() {
		defer func() {
			if r := recover(); r != nil {
				failedSuccessfully = true
			}
		}()
		as.Push(func() {})
	}()

	if !failedSuccessfully {
		t.Fail()
	}
}
