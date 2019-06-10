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
	"os"
	"os/exec"
	"path/filepath"

	log "github.com/cihub/seelog"
	"github.com/magefile/mage/sh"

	"github.com/mysteriumnetwork/node/ci/env"
	"github.com/mysteriumnetwork/node/ci/storage"
	"github.com/mysteriumnetwork/node/logconfig"
)

// PackageLinuxAmd64 builds and stores linux amd64 package
func PackageLinuxAmd64() error {
	logconfig.Bootstrap()
	defer log.Flush()
	if err := packageStandalone("build/myst/myst_linux_amd64", "linux", "amd64"); err != nil {
		return err
	}
	return env.IfRelease(storage.UploadArtifacts)
}

// PackageLinuxArm builds and stores linux arm package
func PackageLinuxArm() error {
	logconfig.Bootstrap()
	defer log.Flush()
	if err := packageStandalone("build/myst/myst_linux_arm", "linux", "arm"); err != nil {
		return err
	}
	return env.IfRelease(storage.UploadArtifacts)
}

// PackageLinuxDebianAmd64 builds and stores debian amd64 package
func PackageLinuxDebianAmd64() error {
	logconfig.Bootstrap()
	defer log.Flush()
	if err := goGet("github.com/debber/debber-v0.3/cmd/debber"); err != nil {
		return err
	}
	if err := sh.RunV("bin/build"); err != nil {
		return err
	}
	if err := packageDebian("build/myst/myst", "amd64"); err != nil {
		return err
	}
	return env.IfRelease(storage.UploadArtifacts)
}

// PackageLinuxDebianArm builds and stores debian arm package
func PackageLinuxDebianArm() error {
	logconfig.Bootstrap()
	defer log.Flush()
	if err := goGet("github.com/debber/debber-v0.3/cmd/debber"); err != nil {
		return err
	}
	if err := sh.RunV("bin/build_xgo", "linux/arm"); err != nil {
		return err
	}
	if err := packageDebian("build/myst/myst_linux_arm", "armhf"); err != nil {
		return err
	}
	return env.IfRelease(storage.UploadArtifacts)
}

// PackageOsxAmd64 builds and stores OSX amd64 package
func PackageOsxAmd64() error {
	logconfig.Bootstrap()
	defer log.Flush()
	if err := packageStandalone("build/myst/myst_darwin_amd64", "darwin", "amd64"); err != nil {
		return err
	}
	return env.IfRelease(storage.UploadArtifacts)
}

// PackageWindowsAmd64 builds and stores Windows amd64 package
func PackageWindowsAmd64() error {
	logconfig.Bootstrap()
	defer log.Flush()
	if err := packageStandalone("build/myst/myst_windows_amd64.exe", "windows", "amd64"); err != nil {
		return err
	}
	return env.IfRelease(storage.UploadArtifacts)
}

// PackageIOS builds and stores iOS package
func PackageIOS() error {
	logconfig.Bootstrap()
	defer log.Flush()
	if err := sh.RunV("bin/package_ios", "amd64"); err != nil {
		return err
	}
	return env.IfRelease(storage.UploadArtifacts)
}

// PackageAndroid builds and stores Android package
func PackageAndroid() error {
	logconfig.Bootstrap()
	defer log.Flush()
	if err := sh.RunV("bin/package_android", "amd64"); err != nil {
		return err
	}
	return env.IfRelease(storage.UploadArtifacts)
}

// PackageDockerAlpine builds and stores docker alpine image
func PackageDockerAlpine() error {
	logconfig.Bootstrap()
	defer log.Flush()
	if err := sh.RunV("bin/package_docker"); err != nil {
		return err
	}
	if err := saveDockerImage("myst:alpine", "build/docker-images/myst_alpine.tgz"); err != nil {
		return err
	}
	return env.IfRelease(storage.UploadDockerImages)
}

// PackageDockerUbuntu builds and stores docker ubuntu image
func PackageDockerUbuntu() error {
	logconfig.Bootstrap()
	defer log.Flush()
	if err := env.EnsureEnvVars(env.BuildVersion); err != nil {
		return err
	}
	if err := sh.RunV("bin/package_docker_ubuntu", env.Str(env.BuildVersion)); err != nil {
		return err
	}
	if err := saveDockerImage("myst:ubuntu", "build/docker-images/myst_ubuntu.tgz"); err != nil {
		return err
	}
	return env.IfRelease(storage.UploadDockerImages)
}

// PackageDockerSwaggerRedoc builds and stores docker swagger redoc image
func PackageDockerSwaggerRedoc() error {
	logconfig.Bootstrap()
	defer log.Flush()
	if err := env.EnsureEnvVars(env.BuildVersion); err != nil {
		return err
	}
	if err := goGet("github.com/go-swagger/go-swagger/cmd/swagger"); err != nil {
		return err
	}
	if err := sh.RunV("bin/swagger_generate"); err != nil {
		return err
	}
	if err := sh.RunV("bin/package_docker_docs"); err != nil {
		return err
	}
	if err := saveDockerImage("tequilapi:"+env.Str(env.BuildVersion), "build/docker-images/tequilapi_redoc.tgz"); err != nil {
		return err
	}
	return env.IfRelease(func() error {
		if err := storage.UploadSingleArtifact("tequilapi.json"); err != nil {
			return err
		}
		if err := storage.UploadDockerImages(); err != nil {
			return err
		}
		return nil
	})
}

func goGet(pkg string) error {
	return sh.RunV("go", "get", "-u", pkg)
}

func packageStandalone(binaryPath, os, arch string) error {
	log.Info("packaging", binaryPath, os, arch)
	envs := map[string]string{
		"BINARY": binaryPath,
	}
	return sh.RunWith(envs, "bin/package_standalone", os, arch)
}

func packageDebian(binaryPath, arch string) error {
	if err := env.EnsureEnvVars(env.BuildVersion); err != nil {
		return err
	}
	envs := map[string]string{
		"BINARY": binaryPath,
	}
	return sh.RunWith(envs, "bin/package_debian", env.Str(env.BuildVersion), arch)
}

func saveDockerImage(image, outputPath string) error {
	parentDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return err
	}
	out, err := os.OpenFile(outputPath, os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		return err
	}
	defer out.Close()

	save := exec.Command("docker", "save", image)
	gzip := exec.Command("gzip")
	gzip.Stdin, _ = save.StdoutPipe()
	gzip.Stdout = out
	if err := gzip.Start(); err != nil {
		return err
	}
	if err := save.Run(); err != nil {
		return err
	}
	if err := gzip.Wait(); err != nil {
		return err
	}
	return nil
}
