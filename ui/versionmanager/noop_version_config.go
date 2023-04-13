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

// NoOpVersionConfig ui version config
type NoOpVersionConfig struct {
}

// NewNoOpVersionConfig constructor for VersionConfig
func NewNoOpVersionConfig() (*NoOpVersionConfig, error) {
	return &NoOpVersionConfig{}, nil
}

// Version returns version to be used
func (vm *NoOpVersionConfig) Version() (string, error) {
	return BundledVersionName, nil
}

func (vm *NoOpVersionConfig) exists() (bool, error) {
	return true, nil
}

func (vm *NoOpVersionConfig) read() (nodeUIVersion, error) {
	return nodeUIVersion{VersionName: ""}, nil
}

func (vm *NoOpVersionConfig) whichFilePath() string {
	return ""
}

func (vm *NoOpVersionConfig) uiDistPath(versionName string) string {
	return ""
}

// UIBuildPath build path to the assets of provided versionName
func (vm *NoOpVersionConfig) UIBuildPath(versionName string) string {
	return ""
}

func (vm *NoOpVersionConfig) uiDistFile(versionName string) string {
	return ""
}

func (vm *NoOpVersionConfig) uiDir() string {
	return ""
}

func (vm *NoOpVersionConfig) write(w nodeUIVersion) error {
	return nil
}
