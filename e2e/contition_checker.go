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

package e2e

import (
	"fmt"
	"time"
)

type conditionChecker func() (bool, error)

func waitForCondition(checkFunc conditionChecker) error {
	return waitForConditionFor(10*time.Second, checkFunc)
}

func waitForConditionFor(duration time.Duration, checkFunc conditionChecker) error {
	timeBegin := time.Now()
	for {
		state, err := checkFunc()
		durationWait := time.Since(timeBegin)
		switch {
		case err != nil:
			return err
		case state:
			return nil
		case durationWait > duration:
			return fmt.Errorf("state was still false after %s", durationWait)
		case !state:
			time.Sleep(1 * time.Second)
		}
	}
}
