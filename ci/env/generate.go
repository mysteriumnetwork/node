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
	"fmt"
	"os"
	"strconv"
)

const devReleaseVersion = "0.0.0-dev"
const ppaDevReleaseVersion = "0.0.0"

type envVar struct {
	key BuildVar
	val string
}

// GenerateEnvFile for sourcing in other stages
func GenerateEnvFile() error {
	isTag := os.Getenv("BUILD_TAG") != ""
	isSnapshot := os.Getenv("BUILD_BRANCH") == "master" && !isTag
	isPr := !isSnapshot && !isTag
	buildVersion := func() string {
		if isTag {
			return os.Getenv("BUILD_TAG")
		}
		return devReleaseVersion + "-" + os.Getenv("BUILD_COMMIT")
	}()
	ppaVersion := func() string {
		if isTag {
			return os.Getenv("BUILD_TAG")
		}
		return ppaDevReleaseVersion
	}()
	vars := []envVar{
		{TagBuild, strconv.FormatBool(isTag)},
		{SnapshotBuild, strconv.FormatBool(isSnapshot)},
		{PrBuild, strconv.FormatBool(isPr)},
		{BuildVersion, buildVersion},
		{PpaVersion, ppaVersion},
		{BuildNumber, os.Getenv(string(BuildNumber))},
		{GithubOwner, os.Getenv(string(GithubOwner))},
		{GithubRepository, os.Getenv(string(GithubRepository))},
		{GithubSnapshotRepository, os.Getenv(string(GithubSnapshotRepository))},
	}
	return writeEnvVars(vars)
}

func writeEnvVars(vars []envVar) error {
	_ = os.Mkdir("./build", 0755)
	file, err := os.Create("./build/env.sh")
	if err != nil {
		return err
	}
	defer file.Close()
	for _, v := range vars {
		_, err := fmt.Fprintf(file, "export %s=%s;\n", v.key, v.val)
		if err != nil {
			return err
		}
	}
	return nil
}
