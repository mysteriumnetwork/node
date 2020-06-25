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
	"path/filepath"

	"github.com/mysteriumnetwork/node/supervisor/util/winutil"
)

// On Windows default log file location will be under C:\Windows\system32\config\systemprofile\AppData\Local\MystSupervisor\myst_supervisor.log
func defaultLogPath() (string, error) {
	root, err := winutil.AppDataDir()
	if err != nil {
		return "", fmt.Errorf("could not get root win directory for logs: %w", err)
	}
	return filepath.Join(root, "myst_supervisor"), nil
}
