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

package dto

import (
	"encoding/json"
	"testing"

	"github.com/mysteriumnetwork/node/datasize"
	"github.com/mysteriumnetwork/node/testkit/assertkit"
	"github.com/stretchr/testify/assert"
)

func TestBandwidthSerialize(t *testing.T) {
	var tests = []struct {
		model        Bandwidth
		expectedJSON string
	}{
		{Bandwidth(1 * datasize.Bit), "1"},
		{Bandwidth(1 * datasize.B), "8"},
		{Bandwidth(0.5 * datasize.B), "4"},
		{Bandwidth(0.51 * datasize.B), "4"},
		{Bandwidth(0 * datasize.Bit), "0"},
		{Bandwidth(1 * datasize.TiB), "8796093022208"},
		{Bandwidth(2 * datasize.TiB), "17592186044416"},
	}

	for _, test := range tests {
		jsonBytes, err := json.Marshal(test.model)

		assert.Nil(t, err)
		assert.Equal(t, test.expectedJSON, string(jsonBytes))
	}
}

func TestBandwidthUnserialize(t *testing.T) {
	var tests = []struct {
		json             string
		expectedModel    Bandwidth
		expectedErrorMsg string
	}{
		{"1", Bandwidth(1 * datasize.Bit), ""},
		{"8", Bandwidth(1 * datasize.B), ""},
		{"4", Bandwidth(0.5 * datasize.B), ""},
		{"8796093022208", Bandwidth(1 * datasize.TiB), ""},
		{"17592186044416", Bandwidth(2 * datasize.TiB), ""},
		{
			"-1",
			Bandwidth(0),
			`strconv.ParseUint: parsing "-1": invalid syntax`,
		},
		{
			"4.08",
			Bandwidth(0),
			`strconv.ParseUint: parsing "4.08": invalid syntax`,
		},
		{
			"1bit",
			Bandwidth(0),
			`invalid character 'b' after top-level value`,
		},
	}

	for _, test := range tests {
		var model Bandwidth
		err := json.Unmarshal([]byte(test.json), &model)

		assert.Equal(t, test.expectedModel, model)
		assertkit.EqualOptionalError(t, err, test.expectedErrorMsg)
	}
}
