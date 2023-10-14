/*
 * Copyright (C) 2021 The "MysteriumNetwork/node" Authors.
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

package versionmanager

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const (
	fileName = "which.json"

	// BundledVersionName bundled node UI version name
	BundledVersionName = "bundled"
)

// NodeUIVersionConfig interface
type NodeUIVersionConfig interface {
	Version() (string, error)
	UIBuildPath(versionName string) string
	uiDir() string
	uiDistPath(versionName string) string
	uiDistFile(versionName string) string
	write(w nodeUIVersion) error
}

// VersionConfig ui version config
type VersionConfig struct {
	nodeUIDir string
}

// NewVersionConfig constructor for VersionConfig
func NewVersionConfig(nodeUIDir string) (*VersionConfig, error) {
	err := os.MkdirAll(nodeUIDir, 0700)
	if err != nil {
		return nil, err
	}

	return &VersionConfig{nodeUIDir: nodeUIDir}, nil
}

// Version returns version to be used
func (vm *VersionConfig) Version() (string, error) {
	exists, err := vm.exists()

	if err != nil {
		return BundledVersionName, err
	}

	if !exists {
		return BundledVersionName, nil
	}

	w, err := vm.read()
	return w.VersionName, err
}

func (vm *VersionConfig) exists() (bool, error) {
	_, err := os.Stat(vm.whichFilePath())

	if err != nil && os.IsPermission(err) {
		return false, err
	}

	if err != nil && os.IsNotExist(err) {
		return false, nil
	}

	if err != nil {
		return false, err
	}

	return true, nil
}

func (vm *VersionConfig) read() (nodeUIVersion, error) {
	data, err := os.ReadFile(vm.whichFilePath())
	if err != nil {
		return nodeUIVersion{}, err
	}

	var w nodeUIVersion
	err = json.Unmarshal(data, &w)
	if err != nil {
		return nodeUIVersion{}, err
	}

	return w, nil
}

func (vm *VersionConfig) whichFilePath() string {
	return filepath.Join(vm.nodeUIDir, fileName)
}

func (vm *VersionConfig) uiDistPath(versionName string) string {
	return filepath.Join(vm.nodeUIDir, versionName)
}

// UIBuildPath build path to the assets of provided versionName
func (vm *VersionConfig) UIBuildPath(versionName string) string {
	return filepath.Join(vm.nodeUIDir, versionName, "build")
}

func (vm *VersionConfig) uiDistFile(versionName string) string {
	return filepath.Join(vm.uiDistPath(versionName), nodeUIAssetName)
}

func (vm *VersionConfig) write(w nodeUIVersion) error {
	configJSON, err := json.Marshal(w)
	if err != nil {
		return err
	}

	return os.WriteFile(vm.whichFilePath(), configJSON, 0644)
}

func (vm *VersionConfig) uiDir() string {
	return vm.nodeUIDir
}

type nodeUIVersion struct {
	VersionName string `json:"version_name"`
}
