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
	"fmt"

	"github.com/magefile/mage/sh"
	"github.com/mysteriumnetwork/go-ci/env"
	"github.com/mysteriumnetwork/go-ci/job"
	"github.com/mysteriumnetwork/node/ci/storage"
	"github.com/mysteriumnetwork/node/logconfig"
)

const (
	// MAVEN_USER user part of token for Sonatype.
	MAVEN_USER = env.BuildVar("MAVEN_USER")
	// MAVEN_PASS password part of token for Sonatype.
	MAVEN_PASS = env.BuildVar("MAVEN_PASS")

	// REPOSITORY_ID references reposity ID in mvn.settings.
	REPOSITORY_ID = "ossrh"
	// REPOSITORY_URL URL for uploading the artifacts.
	REPOSITORY_URL = "https://oss.sonatype.org/service/local/staging/deploy/maven2/"
)

// ReleaseAndroidSDK releases tag Android SDK to maven repo.
func ReleaseAndroidSDK() error {
	logconfig.Bootstrap()

	if err := env.EnsureEnvVars(env.TagBuild, env.BuildVersion); err != nil {
		return err
	}
	job.Precondition(func() bool {
		return env.Bool(env.TagBuild)
	})

	if err := storage.DownloadArtifacts(); err != nil {
		return err
	}

	artifactBaseName := fmt.Sprintf("build/package/mobile-node-%s", env.Str(env.BuildVersion))

	// Deploy AAR (android archive)
	if err := sh.RunWithV(map[string]string{
		"MAVEN_USER": env.Str(MAVEN_USER),
		"MAVEN_PASS": env.Str(MAVEN_PASS),
	}, "mvn",
		"org.apache.maven.plugins:maven-gpg-plugin:3.1.0:sign-and-deploy-file",
		"--settings=bin/package/android/mvn.settings",
		"-DrepositoryId="+REPOSITORY_ID,
		"-Durl="+REPOSITORY_URL,
		"-DpomFile="+artifactBaseName+".pom",
		"-Dfile="+artifactBaseName+".aar",
		"-Dpackaging=aar",
	); err != nil {
		return err
	}

	// Deploy sources JAR
	if err := sh.RunWithV(map[string]string{
		"MAVEN_USER": env.Str(MAVEN_USER),
		"MAVEN_PASS": env.Str(MAVEN_PASS),
	}, "mvn",
		"org.apache.maven.plugins:maven-gpg-plugin:3.1.0:sign-and-deploy-file",
		"--settings=bin/package/android/mvn.settings",
		"-DrepositoryId="+REPOSITORY_ID,
		"-Durl="+REPOSITORY_URL,
		"-DpomFile="+artifactBaseName+".pom",
		"-Dfile="+artifactBaseName+"-sources.jar",
		"-Dpackaging=jar",
		"-Dclassifier=sources",
	); err != nil {
		return err
	}

	return nil
}

// ReleaseAndroidProviderSDK releases tag Android Provider SDK to maven repo.
func ReleaseAndroidProviderSDK() error {
	logconfig.Bootstrap()

	if err := env.EnsureEnvVars(env.TagBuild, env.BuildVersion); err != nil {
		return err
	}
	job.Precondition(func() bool {
		return env.Bool(env.TagBuild)
	})

	if err := storage.DownloadArtifacts(); err != nil {
		return err
	}

	artifactBaseName := fmt.Sprintf("build/package/provider-mobile-node-%s", env.Str(env.BuildVersion))

	// Deploy AAR (android archive)
	if err := sh.RunWithV(map[string]string{
		"MAVEN_USER": env.Str(MAVEN_USER),
		"MAVEN_PASS": env.Str(MAVEN_PASS),
	}, "mvn",
		"org.apache.maven.plugins:maven-gpg-plugin:3.1.0:sign-and-deploy-file",
		"--settings=bin/package/android_provider/mvn.settings",
		"-DrepositoryId="+REPOSITORY_ID,
		"-Durl="+REPOSITORY_URL,
		"-DpomFile="+artifactBaseName+".pom",
		"-Dfile="+artifactBaseName+".aar",
		"-Dpackaging=aar",
	); err != nil {
		return err
	}

	// Deploy sources JAR
	if err := sh.RunWithV(map[string]string{
		"MAVEN_USER": env.Str(MAVEN_USER),
		"MAVEN_PASS": env.Str(MAVEN_PASS),
	}, "mvn",
		"org.apache.maven.plugins:maven-gpg-plugin:3.1.0:sign-and-deploy-file",
		"--settings=bin/package/android_provider/mvn.settings",
		"-DrepositoryId="+REPOSITORY_ID,
		"-Durl="+REPOSITORY_URL,
		"-DpomFile="+artifactBaseName+".pom",
		"-Dfile="+artifactBaseName+"-sources.jar",
		"-Dpackaging=jar",
		"-Dclassifier=sources",
	); err != nil {
		return err
	}

	return nil
}
