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
	"path"
	"strings"
	"text/template"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
	"github.com/rs/zerolog/log"

	"github.com/mysteriumnetwork/go-ci/env"
	"github.com/mysteriumnetwork/go-ci/job"
	"github.com/mysteriumnetwork/go-ci/shell"
	"github.com/mysteriumnetwork/node/ci/deb"
	"github.com/mysteriumnetwork/node/ci/storage"
	"github.com/mysteriumnetwork/node/logconfig"
)

// PackageLinuxAmd64 builds and stores linux amd64 package
func PackageLinuxAmd64() error {
	logconfig.Bootstrap()
	if err := packageStandalone("build/myst/myst_linux_amd64", "linux", "amd64", nil); err != nil {
		return err
	}
	return env.IfRelease(storage.UploadArtifacts)
}

// PackageLinuxArm builds and stores linux arm package
func PackageLinuxArm() error {
	logconfig.Bootstrap()
	if err := packageStandalone("build/myst/myst_linux_arm", "linux", "arm", nil); err != nil {
		return err
	}
	return env.IfRelease(storage.UploadArtifacts)
}

// PackageLinuxArmv6l builds and stores linux armv6 package
func PackageLinuxArmv6l() error {
	logconfig.Bootstrap()
	extraEnv := map[string]string{
		"GOARM": "6",
	}
	if err := packageStandalone("build/myst/myst_linux_armv6l", "linux", "arm", extraEnv); err != nil {
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

// PackageLinuxDebianArm builds and stores debian armv7l+ package
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

// PackageLinuxDebianArm64 builds and stores debian armv6l package
func PackageLinuxDebianArmv6l() error {
	logconfig.Bootstrap()
	if err := goGet("github.com/debber/debber-v0.3/cmd/debber"); err != nil {
		return err
	}
	envi := map[string]string{
		"GOOS":   "linux",
		"GOARCH": "arm",
		"GOARM":  "6",
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
	if err := packageStandalone("build/myst/myst_darwin_amd64", "darwin", "amd64", nil); err != nil {
		return err
	}
	return env.IfRelease(storage.UploadArtifacts)
}

// PackageMacOSArm64 builds and stores macOS arm64 package
func PackageMacOSArm64() error {
	logconfig.Bootstrap()
	if err := packageStandalone("build/myst/myst_darwin_arm64", "darwin", "arm64", nil); err != nil {
		return err
	}
	return env.IfRelease(storage.UploadArtifacts)
}

// PackageWindowsAmd64 builds and stores Windows amd64 package
func PackageWindowsAmd64() error {
	logconfig.Bootstrap()
	if err := packageStandalone("build/myst/myst_windows_amd64.exe", "windows", "amd64", nil); err != nil {
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

	buildVersion := env.Str(env.BuildVersion)
	log.Info().Msgf("Package Android SDK version: %s", buildVersion)

	pomFileOut, err := os.Create(fmt.Sprintf("build/package/mobile-node-%s.pom", buildVersion))
	if err != nil {
		return err
	}
	defer pomFileOut.Close()

	err = pomTemplate.Execute(pomFileOut, struct {
		BuildVersion string
	}{
		BuildVersion: buildVersion,
	})
	if err != nil {
		return err
	}

	return env.IfRelease(storage.UploadArtifacts)
}

// PackageAndroidProvider builds and stores Android Provider package
func PackageAndroidProvider() error {
	job.Precondition(func() bool {
		pr, _ := env.IsPR()
		fullBuild, _ := env.IsFullBuild()
		return !pr || fullBuild
	})
	logconfig.Bootstrap()

	if err := sh.RunV("bin/package_android_provider", "amd64"); err != nil {
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
	pomTemplate, err := template.ParseFiles("bin/package/android_provider/mvn.pom")
	if err != nil {
		return err
	}

	buildVersion := env.Str(env.BuildVersion)
	log.Info().Msgf("Package Android Provider SDK version: %s", buildVersion)

	pomFileOut, err := os.Create(fmt.Sprintf("build/package/provider-mobile-node-%s.pom", buildVersion))
	if err != nil {
		return err
	}
	defer pomFileOut.Close()

	err = pomTemplate.Execute(pomFileOut, struct {
		BuildVersion string
	}{
		BuildVersion: buildVersion,
	})
	if err != nil {
		return err
	}

	return env.IfRelease(storage.UploadArtifacts)
}

func binFmtSupport() error {
	return sh.RunV("docker", "run", "--rm", "--privileged", "linuxkit/binfmt:v0.8")
}

func makeCacheRef(cacheRepo string) string {
	return cacheRepo + ":build-cache"
}

func buildDockerImage(dockerfile string, buildArgs map[string]string, cacheRepo string, tags []string, push bool, platforms []string) error {
	mg.Deps(binFmtSupport)

	if platforms == nil {
		platforms = []string{
			"linux/amd64",
			"linux/arm64",
			"linux/arm",
		}
	}

	args := []string{
		"docker", "buildx", "build",
		"--file", dockerfile,
		"--platform", strings.Join(platforms, ","),
		"--output", fmt.Sprintf("type=image,push=%v", push),
	}
	for buildArgKey, buildArgValue := range buildArgs {
		args = append(args, "--build-arg", buildArgKey+"="+buildArgValue)
	}
	for _, tag := range tags {
		args = append(args, "--tag", tag)
	}

	if cacheRepo != "" {
		args = append(args, "--cache-to=type=registry,mode=max,ref="+makeCacheRef(cacheRepo))
		args = append(args, "--cache-from=type=registry,ref="+makeCacheRef(cacheRepo))
	}

	args = append(args, ".")

	return sh.RunV(args[0], args[1:]...)
}

// BuildMystAlpineImage wraps buildDockerImage with required
// parameters to build myst on Alpine
func BuildMystAlpineImage(tags []string, push bool) error {
	return buildDockerImage(
		path.Join("bin", "docker", "alpine", "Dockerfile"),
		map[string]string{
			string(env.BuildBranch):  env.Str(env.BuildBranch),
			string(env.BuildCommit):  env.Str(env.BuildCommit),
			string(env.BuildNumber):  env.Str(env.BuildNumber),
			string(env.BuildVersion): env.Str(env.BuildVersion),
		},
		"mysteriumnetwork/myst",
		tags,
		push,
		nil,
	)
}

// BuildMystDocumentationImage wraps buildDockerImage with required
// parameters to build TequilAPI ReDoc image
func BuildMystDocumentationImage(tags []string, push bool) error {
	return buildDockerImage(
		path.Join("bin", "docs_docker", "Dockerfile"),
		nil,
		"mysteriumnetwork/documentation",
		tags,
		push,
		[]string{
			"linux/amd64",
		},
	)
}

// PackageDockerAlpine builds and stores docker alpine image
func PackageDockerAlpine() error {
	logconfig.Bootstrap()
	return BuildMystAlpineImage([]string{"myst:alpine"}, false)
}

// PackageDockerSwaggerRedoc builds and stores docker swagger redoc image
func PackageDockerSwaggerRedoc() error {
	logconfig.Bootstrap()

	err := BuildMystDocumentationImage([]string{"tequilapi"}, false)
	if err != nil {
		return err
	}

	return env.IfRelease(func() error {
		return storage.UploadSingleArtifact("tequilapi/docs/swagger.json")
	})
}

func goGet(pkg string) error {
	return sh.RunWith(map[string]string{"GO111MODULE": "off"}, "go", "get", "-u", pkg)
}

func packageStandalone(binaryPath, os, arch string, extraEnvs map[string]string) error {
	log.Info().Msgf("Packaging %s %s %s", binaryPath, os, arch)
	var err error
	if os == "linux" {
		filename := path.Base(binaryPath)
		binaryPath = path.Join("build", filename, filename)
		err = buildBinaryFor(path.Join("cmd", "mysterium_node", "mysterium_node.go"), filename, os, arch, extraEnvs, true)
	} else {
		err = buildCrossBinary(os, arch)
	}
	if err != nil {
		return err
	}

	err = buildBinaryFor(path.Join("cmd", "supervisor", "supervisor.go"), "myst_supervisor", os, arch, extraEnvs, true)
	if err != nil {
		return err
	}

	envs := map[string]string{
		"BINARY": binaryPath,
	}
	return sh.RunWith(envs, "bin/package_standalone", os)
}

func packageDebian(binaryPath, arch string) error {
	if err := env.EnsureEnvVars(env.BuildVersion); err != nil {
		return err
	}
	envs := map[string]string{
		"BINARY": binaryPath,
	}

	if err := deb.TermsTemplateFile("bin/package/installation/templates"); err != nil {
		return err
	}

	return sh.RunWith(envs, "bin/package_debian", env.Str(env.BuildVersion), arch)
}
