/*
 * Copyright (C) 2020 The "MysteriumNetwork/node" Authors.
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

package mbtime

import (
	"testing"
	"time"
)

func TestNow(t *testing.T) {
	for i := 0; i < 100; i++ {
		t1 := Now()
		t2 := Now()
		if t1.Nano() > t2.Nano() {
			t.Fatalf("t1=%d should have been less than or equal to t2=%d", t1, t2)
		}
	}
}

func TestSince(t *testing.T) {
	for i := 0; i < 100; i++ {
		ts := Now()
		d := Since(ts)
		if d < 0 {
			t.Fatalf("d=%d should be greater than or equal to zero", d)
		}
	}
}

func BenchmarkNow(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = Now()
	}
}

func BenchmarkStdNow(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = time.Now()
	}
}

func BenchmarkSince(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ts := Now()
		Since(ts)
	}
}

func BenchmarkStdSince(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ts := time.Now()
		time.Since(ts)
	}
}
