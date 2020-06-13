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

package logconfig

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/windows"
)

// On Windows default log file location will be under C:\Windows\system32\config\systemprofile\AppData\Local\MystSupervisor\myst_supervisor.log
func defaultLogPath() (string, error) {
	root, err := rootWinDirectory()
	if err != nil {
		return "", fmt.Errorf("could not get root win directory for logs: %w", err)
	}
	return filepath.Join(root, "myst_supervisor"), nil
}

func rootWinDirectory() (string, error) {
	root, err := windows.KnownFolderPath(windows.FOLDERID_LocalAppData, windows.KF_FLAG_CREATE)
	if err != nil {
		return "", fmt.Errorf("could not get known local app data folder: %w", err)
	}
	c := filepath.Join(root, "MystSupervisor")
	err = os.MkdirAll(c, os.ModeDir|0700)
	if err != nil {
		return "", fmt.Errorf("could not create logs directory: %w", err)
	}
	return c, nil
}
