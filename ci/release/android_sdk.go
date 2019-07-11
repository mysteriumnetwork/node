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
	cenv "github.com/mysteriumnetwork/go-ci/env"
	"github.com/mysteriumnetwork/go-ci/shell"
	"github.com/mysteriumnetwork/node/ci/env"
	"github.com/mysteriumnetwork/node/ci/storage"
)

// ReleaseAndroidSDK releases Android SDK to sonatype/maven central
func ReleaseAndroidSDK() error {
	err := cenv.EnsureEnvVars(
		cenv.TagBuild,
		cenv.BuildVersion,
		env.SigningGPGKey,
	)
	if err != nil {
		return err
	}
	if !cenv.Bool(cenv.TagBuild) {
		log.Info("not a tag build, skipping ReleaseAndroidSDK action...")
		return nil
	}
	err = storage.DownloadArtifacts()
	if err != nil {
		return err
	}
	err = shell.NewCmdf("gpg --import %s", cenv.Str(env.SigningGPGKey)).Run()
	if err != nil {
		return err
	}
	return shell.NewCmdf("bin/release_android %s", cenv.Str(cenv.BuildVersion)).Run()
}
