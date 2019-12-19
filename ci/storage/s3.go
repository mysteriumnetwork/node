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
	"context"
	"os"
	"path"
	"path/filepath"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/endpoints"
	awsExternal "github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/s3manager"
	"github.com/magefile/mage/sh"
	"github.com/mysteriumnetwork/go-ci/env"
	"github.com/mysteriumnetwork/node/logconfig"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

// Storage wraps AWS S3 client, configures for s3.mysterium.network
// and provides convenience methods
type Storage struct {
	*s3.Client
}

var cacheDir string

const cacheDirPermissions = 0700

func init() {
	var err error
	home, err := os.UserHomeDir()
	if err != nil {
		log.Err(err).Msg("Failed to determine home directory")
		os.Exit(1)
	}
	cacheDir := path.Join(home, ".myst-build-cache")
	err = os.Mkdir(cacheDir, cacheDirPermissions)
	if err != nil && !os.IsExist(err) {
		log.Err(err).Msg("Failed to create storage cache directory")
		os.Exit(1)
	}
}

// NewClient returns *s3.Client, configured to work with https://s3.mysterium.network storage
func NewClient() (*Storage, error) {
	cfg, err := awsExternal.LoadDefaultAWSConfig()
	if err != nil {
		return nil, err
	}
	cfg.EndpointResolver = aws.ResolveWithEndpointURL("https://s3.mysterium.network")
	cfg.Region = endpoints.EuCentral1RegionID
	client := s3.New(cfg)
	client.ForcePathStyle = true
	return &Storage{client}, nil
}

// ListObjects lists objects in storage bucket
func (s *Storage) ListObjects(bucket string) ([]s3.Object, error) {
	req := s.ListObjectsV2Request(&s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
	})
	if err := req.Build(); err != nil {
		return nil, err
	}
	res, err := req.Send(context.Background())
	if err != nil {
		return nil, err
	}
	return res.Contents, nil
}

// FindObject finds an object in storage bucket satisfying given predicate
func (s *Storage) FindObject(bucket string, predicate func(s3.Object) bool) (*s3.Object, error) {
	objects, err := s.ListObjects(bucket)
	if err != nil {
		return nil, err
	}
	for _, obj := range objects {
		if predicate(obj) {
			return &obj, nil
		}
	}
	return nil, nil
}

// GetCacheableFile finds a file in a storage bucket satisfying given predicate. If a local copy with the same size
// does not exist, downloads the file. Otherwise, returns a cached copy.
func (s *Storage) GetCacheableFile(bucket string, predicate func(s3.Object) bool) (string, error) {
	object, err := s.FindObject(bucket, predicate)
	if err != nil {
		return "", errors.Wrap(err, "could not find file in bucket")
	}
	remoteFilename := aws.StringValue(object.Key)
	remoteFileSize := aws.Int64Value(object.Size)

	localFilename := filepath.Join(cacheDir, remoteFilename)
	localFileInfo, err := os.Stat(localFilename)

	var download bool
	switch {
	case err == nil && localFileInfo.Size() != remoteFileSize:
		log.Info().Msgf(
			"Cached copy found: %s, but size mismatched, expected: %d, found: %d",
			localFilename, remoteFileSize, localFileInfo.Size(),
		)
		download = true
	case err != nil && os.IsNotExist(err):
		log.Info().Msgf("Cached copy not found: %s", localFilename)
		download = true
	case err != nil:
		return "", errors.Wrap(err, "error looking up cached copy")
	}

	if download {
		log.Info().Msg("downloading file from the bucket")
		file, err := os.OpenFile(localFilename, os.O_RDWR|os.O_CREATE|os.O_TRUNC, cacheDirPermissions)
		if err != nil {
			return "", err
		}
		defer file.Close()

		downloader := s3manager.NewDownloaderWithClient(s)

		numBytes, err := downloader.Download(file, &s3.GetObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(remoteFilename),
		})
		if err != nil {
			return "", err
		}
		log.Info().Msgf("Downloaded file: %s (%dMB)", localFilename, numBytes/1024/1024)
	} else {
		log.Info().Msg("Returning cached copy")
	}

	return localFilename, nil
}

// MakeBucket creates a bucket in s3 for the build (env.BuildNumber)
func MakeBucket() error {
	logconfig.Bootstrap()
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
	logconfig.Bootstrap()
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

// DownloadDockerImages fetches image archives from s3 bucket
func DownloadDockerImages() error {
	url, err := bucketUrlForBuild()
	if err != nil {
		return err
	}
	return Sync(url+"/docker-images", "build/docker-images")
}

// Sync syncs directories and S3 prefixes.
// Recursively copies new and updated files from the source directory to the destination.
func Sync(source, target string) error {
	if err := sh.RunV("bin/s3", "sync", source, target); err != nil {
		return errors.Wrap(err, "failed to sync artifacts")
	}
	log.Info().Msg("S3 sync successful")
	return nil
}

// Copy copies a local file or S3 object to another location locally or in S3.
func Copy(source, target string) error {
	if err := sh.RunV("bin/s3", "cp", source, target); err != nil {
		return errors.Wrap(err, "failed to copy artifacts")
	}
	log.Info().Msg("S3 copy successful")
	return nil
}

func bucketUrlForBuild() (string, error) {
	if err := env.EnsureEnvVars(env.BuildNumber, env.GithubOwner, env.GithubRepository); err != nil {
		return "", err
	}
	bucket := env.Str(env.GithubOwner) + "-" + env.Str(env.GithubRepository) + "-" + env.Str(env.BuildNumber)
	return "s3://" + bucket, nil
}
