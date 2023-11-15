/*
 * Copyright (C) 2022 The "MysteriumNetwork/node" Authors.
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

package deb

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"text/template"
)

const termsTemplate = `Template: mysterium/terms
Type: text
Description: You have to accept terms and conditions to install this software{{.}}

Template: mysterium/accept_terms
Type: boolean
Description: Do you accept Terms and Conditions?
 In order to install this package you have to accept its terms and conditions
`

// TermsTemplateFile generates terms template file for DEB packages.
func TermsTemplateFile(path string) error {
	templ := template.Must(template.New("terms").Parse(termsTemplate))

	resp, err := http.Get("https://raw.githubusercontent.com/mysteriumnetwork/terms/master/documents/TERMS_NODE_SHORT.md")
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	defer resp.Body.Close()

	terms, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	defer f.Close()

	s := strings.ReplaceAll(string(terms), "\n", "\n ")

	err = templ.Execute(f, strings.ReplaceAll(s, " - ", " . "))
	if err != nil {
		return err
	}

	return nil
}
