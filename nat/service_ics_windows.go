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

package nat

import (
	"golang.org/x/sys/windows/registry"
)

func setICSAddresses(config map[string]string) (map[string]string, error) {
	key, err := registry.OpenKey(registry.LOCAL_MACHINE, `System\CurrentControlSet\Services\SharedAccess\Parameters`, registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		return nil, err
	}
	defer key.Close()

	oldValues := make(map[string]string)
	for k, v := range config {
		s, _, err := key.GetStringValue(k)
		if err != nil {
			if err == registry.ErrNotExist {
				continue
			}
			return nil, err
		}
		oldValues[k] = s

		if err := key.SetStringValue(k, v); err != nil {
			return nil, err
		}
	}

	return oldValues, nil
}
