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

package main

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/pkg/errors"
)

const undefinedDb = ""

var outputDirectory = flag.String("output", "generated", "Name of output source directory with binary data")
var dbFilename = flag.String("dbname", undefinedDb, "Name of the db file to import")
var compress = flag.Bool("compress", false, "Compress data before storing")

func main() {
	flag.Parse()
	if *dbFilename == undefinedDb {
		fmt.Println("dbname parameter expected")
		os.Exit(1)
	}

	binaryData, err := os.ReadFile(*dbFilename)
	exitOnError(err)
	originalDataSize := len(binaryData)

	if *compress {
		binaryData, err = compressData(binaryData)
		exitOnError(err)
	}

	encodedData, err := encodeToBase64(binaryData)
	exitOnError(err)

	//split into 1mb size parts (1part one file)
	parts := splitToFixedLengthSlices(encodedData, 2*1024*1024)
	var dbParts []dbPart
	for i := 0; i < len(parts); i++ {
		dbParts = append(dbParts, newDbPart(i, parts[i]))
	}

	tmpl, err := template.New("dbDataTemplate").Parse(dbPartFileOutput)
	exitOnError(err)
	for partIdx, dbPart := range dbParts {
		dbFile, err := os.Create(filepath.Join(*outputDirectory, fmt.Sprintf("db_part_%d.go", partIdx)))
		exitOnError(err)
		err = tmpl.Execute(dbFile, dbPart)
		exitOnError(err)
	}

	tmpl, err = template.New("dbIndexTemplate").Parse(dbIndexFileOutput)
	exitOnError(err)
	idxFile, err := os.Create(filepath.Join(*outputDirectory, "db_index.go"))
	exitOnError(err)
	err = tmpl.Execute(
		idxFile,
		struct {
			DbParts      []dbPart
			Compressed   bool
			OriginalSize int
		}{
			DbParts:      dbParts,
			Compressed:   *compress,
			OriginalSize: originalDataSize,
		},
	)
	exitOnError(err)
}

func splitToFixedLengthSlices(input []byte, strSize int) [][]byte {
	var res [][]byte

	fullStringCount := len(input) / strSize

	for i := 0; i < fullStringCount; i++ {
		offsetStart := i * strSize
		res = append(res, input[offsetStart:offsetStart+strSize])
	}

	if len(input)%strSize > 0 {
		res = append(res, input[fullStringCount*strSize:])
	}
	return res
}

func encodeToBase64(data []byte) ([]byte, error) {
	encodedDataBuffer := &bytes.Buffer{}
	encodingWriter := base64.NewEncoder(base64.RawStdEncoding, encodedDataBuffer)
	written, err := encodingWriter.Write(data)
	if err != nil {
		return nil, err
	}
	if written != len(data) {
		return nil, errors.New("written and expected data length mismatch")
	}
	encodingWriter.Close()
	return encodedDataBuffer.Bytes(), nil
}

func compressData(data []byte) ([]byte, error) {
	buff := &bytes.Buffer{}
	compressingWriter, err := gzip.NewWriterLevel(buff, gzip.BestCompression)
	if err != nil {
		return nil, err
	}

	written, err := compressingWriter.Write(data)
	if err != nil {
		return nil, err
	}
	if written != len(data) {
		return nil, errors.New("unexpected written and original data size")
	}
	compressingWriter.Close()
	return buff.Bytes(), nil
}

func exitOnError(err error) {
	if err != nil {
		fmt.Println("Error: ", err.Error())
		os.Exit(1)
	}
}

type dbPart struct {
	ID    int
	Lines []string
}

func newDbPart(id int, binaryData []byte) dbPart {
	var lines []string
	for _, binaryPart := range splitToFixedLengthSlices(binaryData, 10*1024*1024) {
		lines = append(lines, string(binaryPart))
	}

	return dbPart{
		ID:    id,
		Lines: lines,
	}
}

var dbPartFileOutput = `/*
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

// generated by generator.go - DO NOT EDIT

const dbDataPart{{.ID}} = {{range .Lines}}"{{.}}" +
	{{end}}""
`

var dbIndexFileOutput = `/*
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

// generated by generator.go - DO NOT EDIT

const originalSize = {{.OriginalSize}}
const dbData = ""{{range .DbParts}} + dbDataPart{{.ID}}{{end}}

// LoadData returns emmbeded database as byte array
func LoadData() ([]byte, error) {
	return EncodedDataLoader(dbData, originalSize, {{.Compressed}})
}
`
