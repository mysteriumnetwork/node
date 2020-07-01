/*
 * Copyright (C) 2020 The "MysteriumNetwork/node" Authors.
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

package winutil

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/windows"
)

func AppDataDir() (string, error) {
	// Default: C:\ProgramData\MystSupervisor
	root, err := windows.KnownFolderPath(windows.FOLDERID_ProgramData, windows.KF_FLAG_CREATE)
	if err != nil {
		return "", fmt.Errorf("could not get known local app data folder: %w", err)
	}
	c := filepath.Join(root, "MystSupervisor")
	err = os.MkdirAll(c, os.ModeDir|0700)
	if err != nil {
		return "", fmt.Errorf("could not create appdata directory: %w", err)
	}
	return c, nil
}
