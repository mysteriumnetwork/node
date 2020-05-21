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
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
	"github.com/mysteriumnetwork/go-ci/env"
	"github.com/mysteriumnetwork/go-ci/job"
	"github.com/mysteriumnetwork/go-ci/shell"
	"github.com/mysteriumnetwork/go-ci/util"
	"github.com/mysteriumnetwork/node/ci/storage"
	"github.com/mysteriumnetwork/node/logconfig"
	"github.com/rs/zerolog/log"
)

// PackageLinuxAmd64 builds and stores linux amd64 package
func PackageLinuxAmd64() error {
	logconfig.Bootstrap()
	if err := packageStandalone("build/myst/myst_linux_amd64", "linux", "amd64"); err != nil {
		return err
	}
	return env.IfRelease(storage.UploadArtifacts)
}

// PackageLinuxArm builds and stores linux arm package
func PackageLinuxArm() error {
	logconfig.Bootstrap()
	if err := packageStandalone("build/myst/myst_linux_arm", "linux", "arm"); err != nil {
		return err
	}
	return env.IfRelease(storage.UploadArtifacts)
}

// PackageLinuxDebianAmd64 builds and stores debian amd64 package
func PackageLinuxDebianAmd64() error {
	logconfig.Bootstrap()
	if err := goGet("github.com/debber/debber-v0.3/cmd/debber"); err != nil {
		return err
	}
	envi := map[string]string{
		"GOOS":   "linux",
		"GOARCH": "amd64",
	}
	if err := sh.RunWith(envi, "bin/build"); err != nil {
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
	if err := goGet("github.com/debber/debber-v0.3/cmd/debber"); err != nil {
		return err
	}
	envi := map[string]string{
		"GOOS":   "linux",
		"GOARCH": "arm",
	}
	if err := sh.RunWith(envi, "bin/build"); err != nil {
		return err
	}
	if err := packageDebian("build/myst/myst", "armhf"); err != nil {
		return err
	}
	return env.IfRelease(storage.UploadArtifacts)
}

// PackageLinuxDebianArm64 builds and stores debian arm64 package
func PackageLinuxDebianArm64() error {
	logconfig.Bootstrap()
	if err := goGet("github.com/debber/debber-v0.3/cmd/debber"); err != nil {
		return err
	}
	envi := map[string]string{
		"GOOS":   "linux",
		"GOARCH": "arm64",
	}
	if err := sh.RunWith(envi, "bin/build"); err != nil {
		return err
	}
	if err := packageDebian("build/myst/myst", "arm64"); err != nil {
		return err
	}
	return env.IfRelease(storage.UploadArtifacts)
}

// PackageMacOSAmd64 builds and stores macOS amd64 package
func PackageMacOSAmd64() error {
	logconfig.Bootstrap()
	if err := packageStandalone("build/myst/myst_darwin_amd64", "darwin", "amd64"); err != nil {
		return err
	}
	return env.IfRelease(storage.UploadArtifacts)
}

// PackageSupervisorMacOSAmd64 builds and stores macOS amd64 supervisor package
func PackageSupervisorMacOSAmd64() error {
	logconfig.Bootstrap()
	if err := packageSupervisor("darwin", "amd64"); err != nil {
		return err
	}
	return env.IfRelease(storage.UploadArtifacts)
}

// PackageWindowsAmd64 builds and stores Windows amd64 package
func PackageWindowsAmd64() error {
	logconfig.Bootstrap()
	if err := packageStandalone("build/myst/myst_windows_amd64.exe", "windows", "amd64"); err != nil {
		return err
	}
	return env.IfRelease(storage.UploadArtifacts)
}

// PackageIOS builds and stores iOS package
func PackageIOS() error {
	job.Precondition(func() bool {
		pr, _ := env.IsPR()
		fullBuild, _ := env.IsFullBuild()
		return !pr || fullBuild
	})
	logconfig.Bootstrap()
	mg.Deps(vendordModules)

	if err := sh.RunV("bin/package_ios", "amd64"); err != nil {
		return err
	}
	return env.IfRelease(storage.UploadArtifacts)
}

// PackageAndroid builds and stores Android package
func PackageAndroid() error {
	job.Precondition(func() bool {
		pr, _ := env.IsPR()
		fullBuild, _ := env.IsFullBuild()
		return !pr || fullBuild
	})
	logconfig.Bootstrap()

	if err := sh.RunV("bin/package_android", "amd64"); err != nil {
		return err
	}

	// Artifacts created by xgo (docker) on CI environment are owned by root.
	// Chown package folder so we can create a POM (see below) in it.
	if _, isCI := os.LookupEnv("CI"); isCI {
		err := shell.NewCmd("sudo chown -R gitlab-runner:gitlab-runner build/package").Run()
		if err != nil {
			return err
		}
	}

	err := env.EnsureEnvVars(env.BuildVersion)
	if err != nil {
		return err
	}
	pomTemplate, err := template.ParseFiles("bin/package/android/mvn.pom")
	if err != nil {
		return err
	}
	pomFileOut, err := os.Create("build/package/mvn.pom")
	if err != nil {
		return err
	}
	defer pomFileOut.Close()

	err = pomTemplate.Execute(pomFileOut, struct {
		BuildVersion string
	}{
		BuildVersion: env.Str(env.BuildVersion),
	})
	if err != nil {
		return err
	}
	return env.IfRelease(storage.UploadArtifacts)
}

// PackageDockerAlpine builds and stores docker alpine image
func PackageDockerAlpine() error {
	logconfig.Bootstrap()
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
	if err := env.EnsureEnvVars(env.BuildVersion); err != nil {
		return err
	}

	if err := sh.RunV("swagger", "generate", "spec", "-o", "tequilapi.json", "--scan-models"); err != nil {
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

// vendordModules uses vend tool to create vendor directory from installed go modules.
// This is a temporary solution needed for ios and android builds since gomobile
// does not support go modules yet and go mod vendor does not include c dependencies.
func vendordModules() error {
	mg.Deps(checkVend)
	return sh.RunV("vend")
}

func checkVend() error {
	path, _ := util.GetGoBinaryPath("vend")
	if path != "" {
		fmt.Println("Tool 'vend' already installed")
		return nil
	}
	err := goGet("github.com/mysteriumnetwork/vend")
	if err != nil {
		fmt.Println("could not go get vend")
		return err
	}
	return nil
}

func goGet(pkg string) error {
	return sh.RunWith(map[string]string{"GO111MODULE": "off"}, "go", "get", "-u", pkg)
}

func packageStandalone(binaryPath, os, arch string) error {
	log.Info().Msgf("Packaging %s %s %s", binaryPath, os, arch)
	if err := buildCrossBinary(os, arch); err != nil {
		return err
	}

	envs := map[string]string{
		"BINARY": binaryPath,
	}
	return sh.RunWith(envs, "bin/package_standalone", os)
}

func packageSupervisor(os, arch string) error {
	log.Info().Msgf("Packaging supervisor %s %s", os, arch)
	envs := map[string]string{
		"GOOS":   os,
		"GOARCH": arch,
	}
	return sh.RunWith(envs, "bin/package_supervisor", os, arch)
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
