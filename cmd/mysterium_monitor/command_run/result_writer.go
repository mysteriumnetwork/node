package command_run

import (
	"encoding/csv"
	"os"

	log "github.com/cihub/seelog"
)

const MYSTERIUM_MONITOR_LOG_PREFIX = "[Mysterium.monitor] "

func NewResultWriter(filePath string) (*resultWriter, error) {
	file, err := os.Create(filePath)
	if err != nil {
		return nil, err
	}

	writer := &resultWriter{
		file:      file,
		csvWriter: csv.NewWriter(file),
	}
	return writer, nil
}

type resultWriter struct {
	file      *os.File
	csvWriter *csv.Writer

	record []string
}

func (writer *resultWriter) NodeStart(nodeKey string) {
	log.Warn(MYSTERIUM_MONITOR_LOG_PREFIX, "Checking node ", nodeKey)

	writer.record = make([]string, 3)
	writer.record[0] = nodeKey
}

func (writer *resultWriter) NodeStatus(status string) {
	log.Warn(MYSTERIUM_MONITOR_LOG_PREFIX, status)

	writer.record[1] = status
	writer.writeRecord()
}

func (writer *resultWriter) NodeError(status string, err error) {
	log.Warn(MYSTERIUM_MONITOR_LOG_PREFIX, status, err)

	writer.record[1] = status
	writer.record[2] = err.Error()
	writer.writeRecord()
}

func (writer *resultWriter) Close() error {
	writer.csvWriter.Flush()
	return writer.file.Close()
}

func (writer *resultWriter) writeRecord() {
	err := writer.csvWriter.Write(writer.record)
	if err != nil {
		panic(err)
	}
}
