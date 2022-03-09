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

package metadata

import (
	"bufio"
	"os"
	"strings"
	"testing"

	"github.com/mysteriumnetwork/terms/terms-go"

	"github.com/stretchr/testify/assert"
)

const notFound = "not found"

func Test_TermsNotCorrupted(t *testing.T) {
	assert.True(t, len(string(terms.TermsNodeShort)) > 1)
	assert.True(t, len(string(terms.TermsBountyPilot)) > 1)
	assert.True(t, len(string(terms.TermsEndUser)) > 1)
	assert.True(t, len(string(terms.TermsExitNode)) > 1)

	assert.Equal(t, moduleVersion(t), terms.TermsVersion)
}

func moduleVersion(t *testing.T) string {
	modFile, err := os.Open("../go.mod")
	assert.NoError(t, err)

	scanner := bufio.NewScanner(modFile)
	for scanner.Scan() {
		if strings.Contains(scanner.Text(), "github.com/mysteriumnetwork/terms") {
			return parseVersionNoPrefix(scanner.Text())
		}
	}
	return notFound
}

func parseVersionNoPrefix(line string) string {
	split := strings.Split(strings.TrimSpace(line), " ")
	for _, s := range split {
		if strings.HasPrefix(s, "v") {
			return s[1:]
		}
	}
	return notFound
}
