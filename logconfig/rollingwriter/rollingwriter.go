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

package rollingwriter

import (
	"fmt"
	"io"
	"os"
	"path"
	"sort"
	"strings"

	"github.com/arthurkiller/rollingwriter"
	"github.com/rs/zerolog/log"
)

// RollingWriter represents logs writer with logs rolling and cleanup support.
type RollingWriter struct {
	config rollingwriter.Config
	Writer io.Writer
}

// NewRollingWriter creates new rolling writer.
func NewRollingWriter(filepath string) (writer *RollingWriter, err error) {
	writer = &RollingWriter{}
	writer.config = rollingwriter.Config{
		TimeTagFormat:     "20060102T150405",
		LogPath:           path.Dir(filepath),
		FileName:          path.Base(filepath),
		RollingPolicy:     rollingwriter.VolumeRolling,
		RollingVolumeSize: "50MB",
		Compress:          true,
		WriterMode:        "lock",
		MaxRemain:         5,
	}
	writer.Writer, err = rollingwriter.NewWriterFromConfig(&writer.config)
	return writer, err
}

// Write writes to underlying rolling writer.
func (w *RollingWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

// CleanObsoleteLogs cleans obsolete logs so that the count of remaining log files is equal to w.config.MaxRemain.
// rollingWriter only handles file rolling at runtime, but if the node is restarted, the count is lost thus we have
// to do this manually.
func (w *RollingWriter) CleanObsoleteLogs() error {
	files, err := os.ReadDir(w.config.LogPath)
	if err != nil {
		return err
	}
	var oldLogFiles []os.FileInfo
	baseFilename := w.config.FileName + ".log"
	for _, file := range files {
		if strings.Contains(file.Name(), baseFilename) && file.Name() != baseFilename {
			fInfo, err := file.Info()
			if err != nil {
				return fmt.Errorf("failed to get file info: %w", err)
			}
			oldLogFiles = append(oldLogFiles, fInfo)
		}
	}
	if len(oldLogFiles) <= w.config.MaxRemain {
		log.Debug().Msgf("Found %d old log files in log directory, skipping cleanup", len(oldLogFiles))
		return nil
	}
	log.Debug().Msgf("Found %d old log files in log directory, proceeding to cleanup", len(oldLogFiles))
	sort.Slice(oldLogFiles, func(i, j int) bool {
		return oldLogFiles[i].ModTime().After(oldLogFiles[j].ModTime())
	})
	for i := w.config.MaxRemain; i < len(oldLogFiles); i++ {
		fp := path.Join(w.config.LogPath, oldLogFiles[i].Name())
		if err := os.Remove(fp); err != nil {
			log.Warn().Err(err).Msg("Failed to remove log file: " + fp)
		}
	}
	return nil
}
