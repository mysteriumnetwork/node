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

import (
	"os"
	"strconv"

	log "github.com/cihub/seelog"
	"github.com/pkg/errors"
)

// RequiredEnvBool reads a mandatory env var to bool
func RequiredEnvBool(v BuildVar) (bool, error) {
	env := os.Getenv(string(v))
	if env == "" {
		return false, errors.New(string(v) + " is not defined")
	}
	b, err := strconv.ParseBool(env)
	if err != nil {
		return false, errors.Wrap(err, "failed to parse env var to bool: "+string(v))
	}
	log.Debug("returning env var (bool)", v)
	return b, nil
}

// RequiredEnvStr reads a mandatory env var to string
func RequiredEnvStr(v BuildVar) (string, error) {
	env := os.Getenv(string(v))
	if env == "" {
		return "", errors.New(string(v) + " is not defined")
	}
	log.Debug("returning env var (str)", v)
	return env, nil
}

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
