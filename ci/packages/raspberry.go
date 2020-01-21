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

package packages

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/mholt/archiver"
	"github.com/mysteriumnetwork/go-ci/env"
	"github.com/mysteriumnetwork/go-ci/job"
	"github.com/mysteriumnetwork/go-ci/shell"
	"github.com/mysteriumnetwork/node/ci/storage"
	"github.com/mysteriumnetwork/node/ci/util/device"
	"github.com/mysteriumnetwork/node/logconfig"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

const (
	raspbianMountPoint = "/mnt/rpi"
	setupDir           = "/home/myst-setup.tmp"
	mountedSetupDir    = raspbianMountPoint + setupDir
)

// PackageLinuxRaspberryImage builds and stores raspberry image
func PackageLinuxRaspberryImage() error {
	job.Precondition(func() bool {
		pr, _ := env.IsPR()
		fullBuild, _ := env.IsFullBuild()
		return !pr || fullBuild
	})
	logconfig.Bootstrap()

	if err := goGet("github.com/debber/debber-v0.3/cmd/debber"); err != nil {
		return err
	}
	if err := shell.NewCmd("bin/build_xgo linux/arm").Run(); err != nil {
		return err
	}
	if err := packageDebian("build/myst/myst_linux_arm", "armhf"); err != nil {
		return err
	}
	if err := buildMystRaspbianImage(); err != nil {
		return err
	}
	return env.IfRelease(storage.UploadArtifacts)
}

func buildMystRaspbianImage() error {
	logconfig.Bootstrap()

	imagePath, err := fetchRaspbianImage()
	if err != nil {
		return err
	}
	if err := configureRaspbianImage(imagePath); err != nil {
		return err
	}
	if err := archiver.DefaultZip.Archive([]string{imagePath}, "build/package/mystberry.zip"); err != nil {
		return err
	}
	if err := os.Remove(imagePath); err != nil {
		return err
	}
	return nil
}

// configureRaspbianImage mounts given raspbian image, spawns a lightweight container via systemd-nspawn and configures it
// see `setup.sh` for setup script executed in container
func configureRaspbianImage(raspbianImagePath string) error {
	envs := map[string]string{
		"PATH":            "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
		"DEBIAN_FRONTEND": "noninteractive",
	}

	if err := shell.NewCmd("sudo apt-get update").Run(); err != nil {
		return err
	}
	if err := shell.NewCmd("sudo apt-get install -y qemu qemu-user-static binfmt-support systemd-container").RunWith(envs); err != nil {
		return err
	}
	loopDevice, err := device.AttachLoop(raspbianImagePath)
	if err != nil {
		return err
	}
	defer func() {
		if err := device.DetachLoop(loopDevice); err != nil {
			log.Warn().Err(err).Msg("")
		}
	}()

	_ = shell.NewCmdf("sudo mkdir -p %s", raspbianMountPoint).Run()
	if err := shell.NewCmdf("sudo mount --options rw %s %s", loopDevice+"p2", raspbianMountPoint).Run(); err != nil {
		return err
	}
	_ = shell.NewCmdf("sudo mkdir -p %s", raspbianMountPoint+"/boot").Run()
	if err := shell.NewCmdf("sudo mount --options rw %s %s", loopDevice+"p1", raspbianMountPoint+"/boot").Run(); err != nil {
		return err
	}
	defer func() {
		if err := shell.NewCmdf("sudo umount --recursive %s", raspbianMountPoint).Run(); err != nil {
			log.Warn().Err(err).Msg("")
		}
	}()
	err = shell.NewCmdf("sudo sed -i s/^/#/g %s", raspbianMountPoint+"/etc/ld.so.preload").Run()
	if err != nil {
		return err
	}
	defer func() {
		err = shell.NewCmdf("sudo sed -i s/^#//g %s", raspbianMountPoint+"/etc/ld.so.preload").Run()
		if err != nil {
			log.Warn().Err(err).Msg("")
		}
	}()

	if err := shell.NewCmdf("sudo cp /usr/bin/qemu-arm-static %s", raspbianMountPoint+"/usr/bin").Run(); err != nil {
		return err
	}

	if err := shell.NewCmdf("sudo mkdir -p %s", mountedSetupDir).Run(); err != nil {
		return err
	}
	if err := shell.NewCmdf("sudo cp build/package/myst_linux_armhf.deb %s", mountedSetupDir).Run(); err != nil {
		return err
	}
	if err := shell.NewCmdf("sudo cp -r bin/package/raspberry/files/. %s", mountedSetupDir).Run(); err != nil {
		return err
	}
	if err := shell.NewCmdf("sudo systemd-nspawn --directory=%s --chdir=%s bash -ev 0-setup-user.sh", raspbianMountPoint, setupDir).Run(); err != nil {
		return err
	}
	if err := shell.NewCmdf("sudo systemd-nspawn --setenv=RELEASE_BUILD=%s --directory=%s --chdir=%s bash -ev 1-setup-node.sh", env.Str(env.TagBuild), raspbianMountPoint, setupDir).Run(); err != nil {
		return err
	}
	if err := shell.NewCmdf("sudo rm -r %s", mountedSetupDir).Run(); err != nil {
		return err
	}
	return nil
}

func fetchRaspbianImage() (filename string, err error) {
	storageClient, err := storage.NewClient()
	if err != nil {
		return "", err
	}

	log.Info().Msg("Looking up Raspbian image file")
	localRaspbianZip, err := storageClient.GetCacheableFile("raspbian", func(object s3.Object) bool {
		return strings.Contains(aws.StringValue(object.Key), "-raspbian-buster-lite")
	})
	if err != nil {
		return "", err
	}

	localRaspbianZipDir, localRaspbianZipFilename := filepath.Split(localRaspbianZip)
	localRaspbianImgDir := filepath.Join(localRaspbianZipDir, localRaspbianZipFilename[0:len(localRaspbianZipFilename)-4])

	log.Info().Msg("Extracting raspbian image to: " + localRaspbianImgDir)
	err = os.RemoveAll(localRaspbianImgDir)
	if err != nil {
		return "", err
	}

	zip := archiver.NewZip()
	zip.OverwriteExisting = true
	err = zip.Unarchive(localRaspbianZip, localRaspbianImgDir)
	if err != nil {
		return "", err
	}

	extractedFiles, err := ioutil.ReadDir(localRaspbianImgDir)
	if err != nil {
		return "", err
	}

	var localRaspbianImg string
	for _, f := range extractedFiles {
		if strings.Contains(f.Name(), ".img") {
			localRaspbianImg = filepath.Join(localRaspbianImgDir, f.Name())
			break
		}
	}
	if localRaspbianImg == "" {
		return "", errors.New("could not find img file in raspbian archive")
	}
	return localRaspbianImg, nil
}
