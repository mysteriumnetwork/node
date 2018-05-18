/*
 * Copyright (C) 2018 The "MysteriumNetwork/node" Authors.
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

package cli

import (
	"fmt"
)

const statusColor = "\033[33m"
const warningColor = "\033[31m"
const successColor = "\033[32m"
const infoColor = "\033[93m"

func status(label string, items ...interface{}) {
	fmt.Printf(statusColor+"[%s] \033[0m", label)
	fmt.Println(items...)
}

func warn(items ...interface{}) {
	fmt.Printf(warningColor + "[WARNING] \033[0m")
	fmt.Println(items...)
}

func success(items ...interface{}) {
	fmt.Printf(successColor + "[SUCCESS] \033[0m")
	fmt.Println(items...)
}

func info(items ...interface{}) {
	fmt.Printf(infoColor + "[INFO] \033[0m")
	fmt.Println(items...)
}
