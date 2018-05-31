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
	"errors"
	"github.com/mysterium/node/datasize"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestBandwidthSerialize(t *testing.T) {
	var tests = []struct {
		model        Bandwidth
		expectedJSON string
	}{
		{Bandwidth(1 * datasize.Bit), "1"},
		{Bandwidth(1 * datasize.Byte), "8"},
		{Bandwidth(0.5 * datasize.Byte), "4"},
		{Bandwidth(0.51 * datasize.Byte), "4"},
		{Bandwidth(0 * datasize.Bit), "0"},
		{Bandwidth(1 * datasize.Terabyte), "8796093022208"},
		{Bandwidth(2 * datasize.Terabyte), "17592186044416"},
	}

	for _, test := range tests {
		jsonBytes, err := json.Marshal(test.model)

		assert.Nil(t, err)
		assert.Equal(t, test.expectedJSON, string(jsonBytes))
	}
}

func TestBandwidthUnserialize(t *testing.T) {
	var tests = []struct {
		json          string
		expectedModel Bandwidth
		expectedError error
	}{
		{"1", Bandwidth(1 * datasize.Bit), nil},
		{"8", Bandwidth(1 * datasize.Byte), nil},
		{"4", Bandwidth(0.5 * datasize.Byte), nil},
		{"8796093022208", Bandwidth(1 * datasize.Terabyte), nil},
		{"17592186044416", Bandwidth(2 * datasize.Terabyte), nil},
		{
			"-1",
			Bandwidth(0),
			errors.New(`strconv.ParseUint: parsing "-1": invalid syntax`),
		},
		{
			"4.08",
			Bandwidth(0),
			errors.New(`strconv.ParseUint: parsing "4.08": invalid syntax`),
		},
		{
			"1bit",
			Bandwidth(0),
			errors.New(`invalid character 'b' after top-level value`),
		},
	}

	for _, test := range tests {
		var model Bandwidth
		err := json.Unmarshal([]byte(test.json), &model)

		assert.Equal(t, test.expectedModel, model)
		if test.expectedError != nil {
			assert.EqualError(t, err, test.expectedError.Error())
		} else {
			assert.NoError(t, err)
		}
	}
}
