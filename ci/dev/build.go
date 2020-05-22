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

package dev

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
	"github.com/mysteriumnetwork/go-ci/env"
	"github.com/mysteriumnetwork/node/logconfig"
	"github.com/mysteriumnetwork/node/utils/fileutil"
	"github.com/rs/zerolog/log"
)

// Build builds the project. Like go tool, it supports cross-platform build with env vars: GOOS, GOARCH.
func Build() error {
	logconfig.Bootstrap()
	if err := buildBinary(path.Join("cmd", "mysterium_node", "mysterium_node.go"), "myst"); err != nil {
		return err
	}
	if err := copyConfig("myst"); err != nil {
		return err
	}
	if err := buildBinary(path.Join("cmd", "supervisor", "supervisor.go"), "myst_supervisor"); err != nil {
		return err
	}
	return nil
}

func linkerFlags() (flags []string) {
	if env.Str(env.BuildBranch) != "" {
		flags = append(flags, "-X", fmt.Sprintf("'github.com/mysteriumnetwork/node/metadata.BuildBranch=%s'", env.Str(env.BuildBranch)))
	}
	if env.Str("BUILD_COMMIT") != "" {
		flags = append(flags, "-X", fmt.Sprintf("'github.com/mysteriumnetwork/node/metadata.BuildCommit=%s'", env.Str("BUILD_COMMIT")))
	}
	if env.Str(env.BuildNumber) != "" {
		flags = append(flags, "-X", fmt.Sprintf("'github.com/mysteriumnetwork/node/metadata.BuildNumber=%s'", env.Str(env.BuildNumber)))
	}
	if env.Str(env.BuildVersion) != "" {
		flags = append(flags, "-X", fmt.Sprintf("'github.com/mysteriumnetwork/node/metadata.Version=%s'", env.Str(env.BuildVersion)))
	}
	return flags
}

func buildBinary(source, target string) error {
	targetOS, ok := os.LookupEnv("GOOS")
	if !ok {
		targetOS = runtime.GOOS
	}
	targetArch, ok := os.LookupEnv("GOARCH")
	if !ok {
		targetArch = runtime.GOARCH
	}
	log.Info().Msgf("Building %s -> %s %s/%s", source, target, targetOS, targetArch)

	buildDir, err := filepath.Abs(path.Join("build", target))
	if err != nil {
		return err
	}

	var flags = []string{"build"}
	if env.Bool("FLAG_RACE") {
		flags = append(flags, "-race")
	}

	ldFlags := linkerFlags()
	flags = append(flags, fmt.Sprintf(`-ldflags=-w -s %s`, strings.Join(ldFlags, " ")))

	if os.Getenv("GOOS") == "windows" {
		target += ".exe"
	}
	flags = append(flags, "-o", path.Join(buildDir, target), source)
	return sh.Run("go", flags...)
}

func copyConfig(target string) error {
	dest, err := filepath.Abs(path.Join("build", target, "config"))
	if err != nil {
		return err
	}

	common, err := filepath.Abs(path.Join("bin", "package", "config", "common"))
	if err != nil {
		return err
	}
	if err := fileutil.CopyDirs(common, dest); err != nil {
		return err
	}

	targetOS, ok := os.LookupEnv("GOOS")
	if !ok {
		targetOS = runtime.GOOS
	}
	osSpecific, err := filepath.Abs(path.Join("bin", "package", "config", targetOS))
	if err := fileutil.CopyDirs(osSpecific, dest); err != nil {
		return err
	}

	return nil
}

// Daemon builds and runs myst daemon
func Daemon() error {
	mg.Deps(Build)

	cmd := "build/myst/myst daemon"
	if runtime.GOOS == "darwin" {
		cmd = "sudo " + cmd
	}
	cmdParts := strings.Split(cmd, " ")
	return sh.RunV(cmdParts[0], cmdParts[1:]...)
}

// Openvpn builds and starts openvpn service with terms accepted
func Openvpn() error {
	mg.Deps(Build)

	cmd := "build/myst/myst service --agreed-terms-and-conditions openvpn"
	if runtime.GOOS == "darwin" {
		cmd = "sudo " + cmd
	}
	cmdParts := strings.Split(cmd, " ")
	return sh.RunV(cmdParts[0], cmdParts[1:]...)
}

// Wireguard builds and starts wireguard service with terms accepted
func Wireguard() error {
	mg.Deps(Build)

	cmd := "build/myst/myst service --agreed-terms-and-conditions wireguard"
	if runtime.GOOS == "darwin" {
		cmd = "sudo " + cmd
	}
	cmdParts := strings.Split(cmd, " ")
	return sh.RunV(cmdParts[0], cmdParts[1:]...)
}

// CLI builds and runs myst CLI
func CLI() error {
	mg.Deps(Build)

	return sh.RunV("build/myst/myst", "cli")
}
