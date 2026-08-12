/*
 * Copyright (C) 2026 The "MysteriumNetwork/node" Authors.
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

package service

import (
	"math"
	"strconv"
	"strings"
)

// byteUnits mirrors the units the runtime backend accepts, so a limit compared
// here is read exactly as the value the backend acted on.
var byteUnits = []struct {
	suffix     string
	multiplier float64
}{
	{"KIB", 1024},
	{"MIB", 1024 * 1024},
	{"GIB", 1024 * 1024 * 1024},
	{"TIB", 1024 * 1024 * 1024 * 1024},
	{"KB", 1000},
	{"MB", 1000 * 1000},
	{"GB", 1000 * 1000 * 1000},
	{"TB", 1000 * 1000 * 1000 * 1000},
	{"B", 1},
}

// parseByteSize reads a size such as "128MiB" into bytes. Comparing sizes this
// way rather than as text means "512MiB", "512 MiB" and "536870912" are the
// one limit they describe, and only a real difference in what a workload may
// use is reported as one.
func parseByteSize(value string) (float64, bool) {
	normalized := strings.TrimSpace(strings.ToUpper(value))
	if normalized == "" {
		return 0, false
	}

	multiplier := float64(1)
	for _, unit := range byteUnits {
		if strings.HasSuffix(normalized, unit.suffix) {
			multiplier = unit.multiplier
			normalized = strings.TrimSpace(strings.TrimSuffix(normalized, unit.suffix))
			break
		}
	}

	parsed, err := strconv.ParseFloat(normalized, 64)
	if err != nil || parsed < 0 {
		return 0, false
	}
	return math.Round(parsed * multiplier), true
}

// parseCPUCores reads a CPU limit such as "0.5" into a number of cores.
func parseCPUCores(value string) (float64, bool) {
	parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	if err != nil || parsed < 0 {
		return 0, false
	}
	return parsed, true
}
