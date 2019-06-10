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
	log "github.com/cihub/seelog"
	"github.com/mholt/archiver"
	"github.com/mysteriumnetwork/node/ci/env"
	"github.com/mysteriumnetwork/node/ci/storage"
	"github.com/mysteriumnetwork/node/ci/util/device"
	"github.com/mysteriumnetwork/node/ci/util/shell"
	"github.com/mysteriumnetwork/node/logconfig"
	"github.com/pkg/errors"
)

const raspbianMountPoint = "/mnt/rpi"

// PackageLinuxRaspberryImage builds and stores raspberry image
func PackageLinuxRaspberryImage() error {
	logconfig.Bootstrap()
	defer log.Flush()
	if err := goGet("github.com/debber/debber-v0.3/cmd/debber"); err != nil {
		return err
	}
	if err := shell.NewCmd("bin/build_xgo linux/arm").Run(); err != nil {
		return err
	}
	if err := packageDebian("build/myst/myst_linux_arm", "armhf"); err != nil {
		return err
	}
	return env.IfRelease(func() error {
		err := buildMystRaspbianImage()
		if err != nil {
			return err
		}
		return storage.UploadArtifacts()
	})
}

func buildMystRaspbianImage() error {
	logconfig.Bootstrap()
	defer log.Flush()

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
	return nil
}

// configureRaspbianImage mounts given raspbian image, spawns a lightweight container via systemd-nspawn and configures it
// see `setup.sh` for setup script executed in container
func configureRaspbianImage(raspbianImagePath string) error {
	envs := map[string]string{
		"PATH":            "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
		"DEBIAN_FRONTEND": "noninteractive",
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
			log.Warn(err)
		}
	}()

	_ = os.MkdirAll(raspbianMountPoint, 0755)
	if err := shell.NewCmdf("sudo mount --options rw %s %s", loopDevice+"p2", raspbianMountPoint).Run(); err != nil {
		return err
	}
	_ = os.MkdirAll(raspbianMountPoint+"/boot", 0755)
	if err := shell.NewCmdf("sudo mount --options rw %s %s", loopDevice+"p1", raspbianMountPoint+"/boot").Run(); err != nil {
		return err
	}
	defer func() {
		if err := shell.NewCmdf("sudo umount --recursive %s", raspbianMountPoint).Run(); err != nil {
			log.Warn(err)
		}
	}()
	if err := shell.NewCmdf("sudo cp /usr/bin/qemu-arm-static %s", raspbianMountPoint+"/usr/bin").Run(); err != nil {
		return err
	}

	setupDir := "/home/pi/myst-setup"
	mountedSetupDir := raspbianMountPoint + setupDir
	if err := shell.NewCmdf("sudo mkdir -p %s", mountedSetupDir).Run(); err != nil {
		return err
	}
	if err := shell.NewCmdf("sudo cp build/package/myst_linux_armhf.deb %s", mountedSetupDir).Run(); err != nil {
		return err
	}
	if err := shell.NewCmdf("sudo cp -r bin/package/raspberry/files/. %s", mountedSetupDir).Run(); err != nil {
		return err
	}
	if err := shell.NewCmdf("sudo systemd-nspawn --directory=%s --chdir=%s bash -ev setup.sh", raspbianMountPoint, setupDir).Run(); err != nil {
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

	log.Infof("looking up raspbian image file")
	localRaspbianZip, err := storageClient.GetCacheableFile("raspbian", func(object s3.Object) bool {
		return strings.Contains(aws.StringValue(object.Key), "-raspbian-stretch-lite")
	})
	if err != nil {
		return "", err
	}

	localRaspbianZipDir, localRaspbianZipFilename := filepath.Split(localRaspbianZip)
	localRaspbianImgDir := filepath.Join(localRaspbianZipDir, localRaspbianZipFilename[0:len(localRaspbianZipFilename)-4])

	log.Infof("extracting raspbian image to %s", localRaspbianImgDir)
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
