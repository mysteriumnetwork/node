/*
 * Copyright (C) 2019 The "MysteriumNetwork/node" Authors.
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

package env

import log "github.com/cihub/seelog"

// IfRelease performs func passed as an arg if current build is any kind of release
func IfRelease(do func() error) error {
	isRelease, err := isRelease()
	if err != nil {
		return err
	}
	if isRelease {
		log.Info("release build detected, performing conditional action")
		return do()
	}
	log.Info("not a release build, skipping conditional action")
	return nil
}

func isRelease() (bool, error) {
	isTag, err := RequiredEnvBool(TagBuild)
	if err != nil {
		return false, err
	}
	isSnapshot, err := RequiredEnvBool(SnapshotBuild)
	if err != nil {
		return false, err
	}
	return isTag || isSnapshot, nil
}
