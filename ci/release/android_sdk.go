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

package release

import (
	log "github.com/cihub/seelog"
	"github.com/mysteriumnetwork/go-ci/env"
	"github.com/mysteriumnetwork/go-ci/shell"
	"github.com/mysteriumnetwork/node/ci/storage"
)

const (
	// ProxySonatypeUsername username for proxy used in publishing to sonatype
	ProxySonatypeUsername = env.BuildVar("PROXY_SONATYPE_USERNAME")
	// ProxySonatypePassword password for proxy used in publishing to sonatype
	ProxySonatypePassword = env.BuildVar("PROXY_SONATYPE_PASSWORD")
	// SonatypeUsername username for sonatype
	SonatypeUsername = env.BuildVar("SONATYPE_USERNAME")
	// SonatypePassword password for sonatype
	SonatypePassword = env.BuildVar("SONATYPE_PASSWORD")
)

// ReleaseAndroidSDK releases Android SDK to sonatype/maven central
func ReleaseAndroidSDK() error {
	err := env.EnsureEnvVars(
		env.TagBuild,
		env.BuildVersion,
	)
	if err != nil {
		return err
	}
	if !env.Bool(env.TagBuild) {
		log.Info("not a tag build, skipping ReleaseAndroidSDK action...")
		return nil
	}
	err = env.EnsureEnvVars(
		ProxySonatypeUsername,
		ProxySonatypePassword,
		SonatypeUsername,
		SonatypePassword,
	)
	if err != nil {
		return err
	}
	err = storage.DownloadArtifacts()
	if err != nil {
		return err
	}

	envs := getEnvs(ProxySonatypeUsername, ProxySonatypePassword, SonatypeUsername, SonatypePassword)
	return shell.NewCmdf("bash -ev bin/release_android %s", env.Str(env.BuildVersion)).RunWith(envs)
}

func getEnvs(vars ...env.BuildVar) map[string]string {
	envs := map[string]string{}
	for _, v := range vars {
		envs[string(v)] = env.Str(v)
	}
	return envs
}
