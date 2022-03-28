/*
 * Copyright (C) 2017 The "MysteriumNetwork/node" Authors.
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

package clio

import (
	"fmt"
	"unicode"
)

const statusColor = "\033[33m"
const warningColor = "\033[31m"
const successColor = "\033[32m"
const infoColor = "\033[93m"

// Status prints a message with a given status.
func Status(label string, items ...interface{}) {
	fmt.Printf(statusColor+"[%s] \033[0m", label)
	fmt.Println(sentenceCase(fmt.Sprintln(items...)))
}

// Warn prints a warning.
func Warn(items ...interface{}) {
	fmt.Printf(warningColor + "[WARNING] \033[0m")
	fmt.Println(sentenceCase(fmt.Sprint(items...)))
}

// Warnf prints a warning using fmt.Printf.
func Warnf(format string, items ...interface{}) {
	fmt.Printf(warningColor + "[WARNING] \033[0m")
	fmt.Print(sentenceCase(fmt.Sprintf(format, items...)))
}

// Success prints a success message.
func Success(items ...interface{}) {
	fmt.Printf(successColor + "[SUCCESS] \033[0m")
	fmt.Println(sentenceCase(fmt.Sprint(items...)))
}

// Info prints an information message.
func Info(items ...interface{}) {
	fmt.Printf(infoColor + "[INFO] \033[0m")
	fmt.Println(sentenceCase(fmt.Sprint(items...)))
}

// Error prints an error message
func Error(items ...interface{}) {
	fmt.Printf(warningColor + "[ERROR] \033[0m")
	fmt.Println(sentenceCase(fmt.Sprint(items...)))
}

// Infof prints an information message using fmt.Printf.
func Infof(format string, items ...interface{}) {
	fmt.Printf(infoColor + "[INFO] \033[0m")
	fmt.Print(sentenceCase(fmt.Sprintf(format, items...)))
}

// sentenceCase capitalizes the first letter.
func sentenceCase(s string) string {
	runes := []rune(s)
	return string(unicode.ToUpper(runes[0])) + string(runes[1:])
}
