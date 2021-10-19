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

package reftracker

import (
	"testing"
	"time"
)

const testPatrolPeriod = 10 * time.Millisecond

func TestCollect(t *testing.T) {
	tr := NewRefTracker(testPatrolPeriod)
	defer tr.Close()

	called := make(chan struct{})

	tr.Put("1", 100*time.Millisecond, func() {
		close(called)
	})

	time.Sleep(200 * time.Millisecond)
	select {
	case <-called:
	default:
		t.Error("release callback was not invoked")
	}
}

func TestUseAndCollect(t *testing.T) {
	tr := NewRefTracker(testPatrolPeriod)
	defer tr.Close()

	called := make(chan struct{})

	tr.Put("1", 100*time.Millisecond, func() {
		close(called)
	})

	time.Sleep(50 * time.Millisecond)

	select {
	case <-called:
		t.Error("release callback was invoked too early")
		return
	default:
	}

	err := tr.Incr("1")
	if err != nil {
		t.Errorf("error: %v", err)
		return
	}

	time.Sleep(200 * time.Millisecond)

	select {
	case <-called:
		t.Error("release callback was invoked too early")
		return
	default:
	}

	tr.Decr("1")

	time.Sleep(200 * time.Millisecond)
	select {
	case <-called:
	default:
		t.Error("release callback was not invoked")
		return
	}
}

func TestNegativeRefcountPanic(t *testing.T) {
	var failedSuccessfully bool
	tr := NewRefTracker(testPatrolPeriod)
	defer tr.Close()

	tr.Put("1", 2*time.Second, func() {})

	func() {
		defer func() {
			if r := recover(); r != nil {
				failedSuccessfully = true
			}
		}()
		tr.Decr("1")
	}()

	if !failedSuccessfully {
		t.Fail()
	}
}

func TestPutIdempotent(t *testing.T) {
	tr := NewRefTracker(testPatrolPeriod)
	defer tr.Close()

	called := make(chan struct{})

	tr.Put("1", 100*time.Millisecond, func() {
		close(called)
	})
	tr.Put("1", 1*time.Millisecond, func() {
		t.Fail()
	})

	time.Sleep(200 * time.Millisecond)
	select {
	case <-called:
	default:
		t.Error("release callback was not invoked")
	}
}
