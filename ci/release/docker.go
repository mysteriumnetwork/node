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
const mystSnapshotsRepo = "mysteriumnetwork/myst-snapshots"

// ReleaseDockerSnapshot uploads docker snapshot images to myst snapshots repo
func ReleaseDockerSnapshot() error {
	logconfig.Bootstrap()
	defer log.Flush()

	if err := env.EnsureEnvVars(env.SnapshotBuild, env.BuildVersion, env.DockerHubPassword, env.DockerHubUsername); err != nil {
		return err
	}

	if !env.Bool(env.SnapshotBuild) {
		log.Info("not a snapshot build, skipping ReleaseSnapshot action...")
		return nil
	}

	if err := storage.DownloadDockerImages(); err != nil {
		return err
	}

	if err := dockerLogin(env.Str(env.DockerHubUsername), env.Str(env.DockerHubPassword)); err != nil {
		return err
	}
	defer dockerLogout()

	dockerImageArchives, err := listDockerImageArchives()
	if err != nil {
		return err
	}

	for _, dockerImageArchive := range dockerImageArchives {
		log.Info("Restoring from: ", dockerImageArchive)
		imageName, err := restoreDockerImage(dockerImageArchive)
		if err != nil {
			return err
		}
		log.Info("Restored image: ", imageName)

		if err := pushDockerImage(imageName, env.Str(env.BuildVersion), mystSnapshotsRepo); err != nil {
			return err
		}

		if err := removeDockerImage(imageName); err != nil {
			return err
		}
	}
	return nil
}

func pushDockerImage(localImageName string, buildVersion string, remoteRepoName string) error {
	parts := strings.Split(localImageName, ":")
	tagName := fmt.Sprintf("%s:%s-%s", remoteRepoName, parts[1], buildVersion)
	// docker and repositories don't like upper case letters in references
	tagName = strings.ToLower(tagName)
	log.Info("Tagging ", localImageName, " as ", tagName)
	if err := exec.Command("docker", "tag", localImageName, tagName).Run(); err != nil {
		return err
	}
	log.Info("Pushing ", tagName, " to remote repo")
	if err := exec.Command("docker", "push", tagName).Run(); err != nil {
		return err
	}
	return removeDockerImage(tagName)
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
