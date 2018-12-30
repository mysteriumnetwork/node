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

package gendb

import (
	"compress/gzip"
	"encoding/base64"
	"errors"
	"io/ioutil"
	"strings"
)

// EncodedDataLoader returns emmbeded database as byte array
func EncodedDataLoader(data string, originalSize int, compressed bool) (decompressed []byte, err error) {
	reader := base64.NewDecoder(base64.RawStdEncoding, strings.NewReader(data))

	if compressed {
		reader, err = gzip.NewReader(reader)
		if err != nil {
			return nil, err
		}
	}

	decompressed, err = ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	if len(decompressed) != originalSize {
		return nil, errors.New("original and decompressed data size mismatch")
	}
	return decompressed, nil
}
