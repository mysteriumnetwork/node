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

package cmdutil

import (
	"os/exec"

	"github.com/mysteriumnetwork/node/utils/stringutil"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

func PowerShell(cmd string) ([]byte, error) {
	log.Debug().Msgf("[powershell] executing: '%s'", cmd)
	out, err := exec.Command("powershell", "-Command", cmd).CombinedOutput()
	if err != nil {
		return nil, errors.Errorf("'powershell -Command %v': %v output: %s", cmd, stringutil.RemoveErrorsAndBOMUTF8(err.Error()), stringutil.RemoveErrorsAndBOMUTF8Byte(out))
	}
	log.Debug().Msgf("[powershell] done: '%s', raw output: '%s'", cmd, out)
	return stringutil.RemoveErrorsAndBOMUTF8Byte(out), nil
}
