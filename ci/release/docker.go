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
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"os/exec"
	"path/filepath"
	"strings"

	log "github.com/cihub/seelog"
	"github.com/mysteriumnetwork/go-ci/env"
	"github.com/mysteriumnetwork/node/ci/storage"
	"github.com/mysteriumnetwork/node/logconfig"
	"github.com/pkg/errors"
)

const dockerImagesDir = "build/docker-images"

type dockerReleasable struct {
	partialLocalName string
	repository       string
	tags             []string
}

type releaseDockerHubOpts struct {
	username    string
	password    string
	releasables []dockerReleasable
}

func releaseDockerHub(opts *releaseDockerHubOpts) error {
	err := dockerLogin(opts.username, opts.password)
	if err != nil {
		return err
	}
	defer dockerLogout()

	err = storage.DownloadDockerImages()
	if err != nil {
		return err
	}
	archives, err := listDockerImageArchives()
	if err != nil {
		return err
	}

	for _, archive := range archives {
		log.Info("Restoring from: ", archive)
		imageName, err := restoreDockerImage(archive)
		if err != nil {
			return err
		}
		log.Info("Restored image: ", imageName)

		var releasable *dockerReleasable
		for _, r := range opts.releasables {
			if strings.Contains(imageName, r.partialLocalName) {
				releasable = &r
				break
			}
		}
		if releasable == nil {
			log.Info("image didn't match any releasable definition, skipping: ", imageName)
			continue
		}

		log.Debug("resolved releasable info: ", releasable)
		for _, tag := range releasable.tags {
			err = pushDockerImage(imageName, releasable.repository, tag)
			if err != nil {
				return err
			}
		}

		err = removeDockerImage(imageName)
		if err != nil {
			return err
		}
	}

	return nil
}

// ReleaseDockerSnapshot uploads docker snapshot images to myst snapshots repository in docker hub
func ReleaseDockerSnapshot() error {
	logconfig.Bootstrap()
	defer log.Flush()

	err := env.EnsureEnvVars(
		env.SnapshotBuild,
		env.BuildVersion,
		env.DockerHubPassword,
		env.DockerHubUsername,
	)
	if err != nil {
		return err
	}

	if !env.Bool(env.SnapshotBuild) {
		log.Info("not a snapshot build, skipping ReleaseDockerSnapshot action...")
		return nil
	}

	releasables := []dockerReleasable{
		{partialLocalName: "myst:alpine", repository: "mysteriumnetwork/myst-snapshots", tags: []string{
			env.Str(env.BuildVersion) + "-alpine",
		}},
		{partialLocalName: "myst:ubuntu", repository: "mysteriumnetwork/myst-snapshots", tags: []string{
			env.Str(env.BuildVersion) + "-ubuntu",
		}},
	}
	return releaseDockerHub(&releaseDockerHubOpts{
		username:    env.Str(env.DockerHubUsername),
		password:    env.Str(env.DockerHubPassword),
		releasables: releasables,
	})
}

// ReleaseDockerTag uploads docker tag release images to docker hub
func ReleaseDockerTag() error {
	logconfig.Bootstrap()
	defer log.Flush()

	err := env.EnsureEnvVars(
		env.TagBuild,
		env.RCBuild,
		env.BuildVersion,
		env.DockerHubPassword,
		env.DockerHubUsername,
	)
	if err != nil {
		return err
	}

	if !env.Bool(env.TagBuild) {
		log.Info("not a tag build, skipping ReleaseDockerTag action...")
		return nil
	}

	var releasables []dockerReleasable
	if env.Bool(env.RCBuild) {
		releasables = []dockerReleasable{
			{partialLocalName: "myst:alpine", repository: "mysteriumnetwork/myst", tags: []string{
				env.Str(env.BuildVersion) + "-alpine",
			}},
			{partialLocalName: "myst:ubuntu", repository: "mysteriumnetwork/myst", tags: []string{
				env.Str(env.BuildVersion) + "-ubuntu",
			}},
			{partialLocalName: "tequilapi:", repository: "mysteriumnetwork/documentation", tags: []string{
				env.Str(env.BuildVersion),
			}},
		}
	} else {
		releasables = []dockerReleasable{
			{partialLocalName: "myst:alpine", repository: "mysteriumnetwork/myst", tags: []string{
				env.Str(env.BuildVersion) + "-alpine",
				"latest-alpine",
				"latest",
			}},
			{partialLocalName: "myst:ubuntu", repository: "mysteriumnetwork/myst", tags: []string{
				env.Str(env.BuildVersion) + "-ubuntu",
				"latest-ubuntu",
			}},
			{partialLocalName: "tequilapi:", repository: "mysteriumnetwork/documentation", tags: []string{
				env.Str(env.BuildVersion),
				"latest",
			}},
		}
	}
	return releaseDockerHub(&releaseDockerHubOpts{
		username:    env.Str(env.DockerHubUsername),
		password:    env.Str(env.DockerHubPassword),
		releasables: releasables,
	})
}

func pushDockerImage(localImageName, repository, tag string) error {
	imageName := fmt.Sprintf("%s:%s", repository, tag)
	// docker and repositories don't like upper case letters in references
	imageName = strings.ToLower(imageName)
	log.Info("Tagging ", localImageName, " as ", imageName)
	if err := exec.Command("docker", "tag", localImageName, imageName).Run(); err != nil {
		return errors.Wrapf(err, "error tagging docker image %q as %q", localImageName, imageName)
	}
	log.Info("Pushing ", imageName, " to remote repository")
	if err := exec.Command("docker", "push", imageName).Run(); err != nil {
		return errors.Wrapf(err, "error pushing docker image %q", imageName)
	}
	return removeDockerImage(imageName)
}

func dockerLogin(username, password string) error {
	err := exec.Command("docker", "login", "-u", username, "-p", password).Run()
	return errors.Wrap(err, "error logging into docker")
}

func dockerLogout() {
	if err := exec.Command("docker", "logout").Run(); err != nil {
		log.Warn("error logging out from docker: ", err)
	}
}

func listDockerImageArchives() ([]string, error) {
	files, err := ioutil.ReadDir(dockerImagesDir)
	if err != nil {
		return nil, err
	}
	var archives []string
	for _, dockerArchiveFile := range files {
		if strings.HasSuffix(dockerArchiveFile.Name(), ".tgz") {
			archives = append(archives, filepath.Join(dockerImagesDir, dockerArchiveFile.Name()))
		}
	}
	return archives, nil
}

func restoreDockerImage(archiveFile string) (restoredImage string, err error) {
	catFile := exec.Command("cat", archiveFile)
	decompress := exec.Command("gzip", "-d")
	dockerLoad := exec.Command("docker", "load")

	decompress.Stdin, err = catFile.StdoutPipe()
	if err != nil {
		return "", err
	}
	dockerLoad.Stdin, err = decompress.StdoutPipe()
	if err != nil {
		return "", err
	}
	var dockerOutput bytes.Buffer
	dockerLoad.Stdout = &dockerOutput

	if err = decompress.Start(); err != nil {
		return "", err
	}

	if err = dockerLoad.Start(); err != nil {
		return "", err
	}

	if err = catFile.Run(); err != nil {
		return "", err
	}
	if err = dockerLoad.Wait(); err != nil {
		return "", err
	}
	scanner := bufio.NewScanner(&dockerOutput)
	var lastLine string
	for scanner.Scan() {
		lastLine = scanner.Text()
	}
	if scanner.Err() != nil {
		return "", scanner.Err()
	}
	parts := strings.SplitN(lastLine, ":", 2)
	imageName := strings.TrimLeft(parts[1], " ")
	return imageName, nil
}

func removeDockerImage(imageName string) error {
	log.Info("Removing: ", imageName)
	err := exec.Command("docker", "image", "rm", imageName).Run()
	return errors.Wrapf(err, "error removing docker image %q", imageName)
}
