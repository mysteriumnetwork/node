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

package storage

import (
	log "github.com/cihub/seelog"
	"github.com/magefile/mage/sh"
	"github.com/mysteriumnetwork/node/ci/env"
	"github.com/pkg/errors"
)

// MakeBucket creates a bucket in s3 for the build (env.BuildNumber)
func MakeBucket() error {
	return env.IfRelease(func() error {
		url, err := bucketUrlForBuild()
		if err != nil {
			return err
		}
		return sh.RunV("bin/s3", "mb", url)
	})
}

// RemoveBucket removes bucket
func RemoveBucket() error {
	return env.IfRelease(func() error {
		url, err := bucketUrlForBuild()
		if err != nil {
			return err
		}
		return sh.RunV("bin/s3", "rb", "--force", url)
	})
}

// UploadArtifacts uploads all artifacts to s3 build bucket
func UploadArtifacts() error {
	url, err := bucketUrlForBuild()
	if err != nil {
		return err
	}
	return Sync("build/package", url+"/build-artifacts")
}

// UploadDockerImages uploads all docker images to s3 build bucket
func UploadDockerImages() error {
	url, err := bucketUrlForBuild()
	if err != nil {
		return err
	}
	return Sync("build/docker-images", url+"/docker-images")
}

// UploadSingleArtifact uploads a single file to s3 build bucket
func UploadSingleArtifact(path string) error {
	url, err := bucketUrlForBuild()
	if err != nil {
		return err
	}
	return Copy(path, url+"/build-artifacts/")
}

// DownloadArtifacts downloads all artifacts from s3 build bucket
func DownloadArtifacts() error {
	url, err := bucketUrlForBuild()
	if err != nil {
		return err
	}
	return Sync(url+"/build-artifacts", "build/package")
}

// Sync syncs directories and S3 prefixes.
// Recursively copies new and updated files from the source directory to the destination.
func Sync(source, target string) error {
	if err := sh.RunV("bin/s3", "sync", source, target); err != nil {
		return errors.Wrap(err, "failed to sync artifacts")
	}
	log.Info("s3 sync successful")
	return nil
}

// Copy copies a local file or S3 object to another location locally or in S3.
func Copy(source, target string) error {
	if err := sh.RunV("bin/s3", "cp", source, target); err != nil {
		return errors.Wrap(err, "failed to copy artifacts")
	}
	log.Info("s3 copy successful")
	return nil
}

func bucketUrlForBuild() (string, error) {
	buildNumber, err := env.RequiredEnvStr(env.BuildNumber)
	if err != nil {
		return "", err
	}
	owner, err := env.RequiredEnvStr(env.GithubOwner)
	if err != nil {
		return "", err
	}
	projectName, err := env.RequiredEnvStr(env.GithubRepository)
	if err != nil {
		return "", err
	}
	bucket := owner + "-" + projectName + "-" + buildNumber
	return "s3://" + bucket, nil
}
