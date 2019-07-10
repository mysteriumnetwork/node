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
)

const dockerImagesDir = "build/docker-images"

type dockerReleasable struct {
	partialLocalName string
	repository       string
	tags             []string
}

func releaseDockerHub(username, password string, releasables []dockerReleasable) error {
	err := dockerLogin(username, password)
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
		for _, r := range releasables {
			if !strings.Contains(imageName, r.partialLocalName) {
				continue
			}
			releasable = &r
		}
		if releasable == nil {
			log.Info("Image didn't match any releasable definition, skipping: ", imageName)
			continue
		}

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

// ReleaseDockerSnapshot uploads docker snapshot images to myst snapshots repo in docker hub
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
	return releaseDockerHub(env.Str(env.DockerHubUsername), env.Str(env.DockerHubPassword), releasables)
}

func pushDockerImage(localImageName string, buildVersion string, remoteRepoName string) error {
	parts := strings.Split(localImageName, ":")
	tagName := fmt.Sprintf("%s:%s-%s", remoteRepoName, parts[1], buildVersion)
	// docker and repositories don't like upper case letters in references
	imageName = strings.ToLower(imageName)
	log.Info("Tagging ", localImageName, " as ", imageName)
	if err := exec.Command("docker", "tag", localImageName, imageName).Run(); err != nil {
		return err
	}
	log.Info("Pushing ", imageName, " to remote repo")
	if err := exec.Command("docker", "push", imageName).Run(); err != nil {
		return err
	}
	return removeDockerImage(imageName)
}

func dockerLogin(username, password string) error {
	return exec.Command("docker", "login", "-u", username, "-p", password).Run()
}

func dockerLogout() {
	if err := exec.Command("docker", "logout").Run(); err != nil {
		log.Warn("Error loging out from docker: ", err)
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
	return exec.Command("docker", "image", "rm", imageName).Run()
}
