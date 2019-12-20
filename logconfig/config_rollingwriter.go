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

package logconfig

import (
	"io"
	"io/ioutil"
	"os"
	"path"
	"sort"
	"strings"

	"github.com/arthurkiller/rollingwriter"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type rollingWriter struct {
	config rollingwriter.Config
	fw     io.Writer
}

func newRollingWriter(opts *LogOptions) (writer *rollingWriter, err error) {
	writer = &rollingWriter{}
	writer.config = rollingwriter.Config{
		TimeTagFormat:     "20060102T150405",
		LogPath:           path.Dir(opts.Filepath),
		FileName:          path.Base(opts.Filepath),
		RollingPolicy:     rollingwriter.VolumeRolling,
		RollingVolumeSize: "50MB",
		Compress:          true,
		WriterMode:        "lock",
		MaxRemain:         5,
	}
	writer.fw, err = rollingwriter.NewWriterFromConfig(&writer.config)
	return writer, err
}

// cleanObsoleteLogs cleans obsolete logs so that the count of remaining log files is equal to w.config.MaxRemain.
// rollingWriter only handles file rolling at runtime, but if the node is restarted, the count is lost thus we have
// to do this manually.
func (w *rollingWriter) cleanObsoleteLogs() error {
	files, err := ioutil.ReadDir(w.config.LogPath)
	if err != nil {
		return err
	}
	var oldLogFiles []os.FileInfo
	baseFilename := w.config.FileName + ".log"
	for _, file := range files {
		if strings.Contains(file.Name(), baseFilename) && file.Name() != baseFilename {
			oldLogFiles = append(oldLogFiles, file)
		}
	}
	if len(oldLogFiles) <= w.config.MaxRemain {
		log.Debug().Msgf("Found %d old log files in log directory, skipping cleanup", len(oldLogFiles))
		return nil
	}
	log.Info().Msgf("Found %d old log files in log directory, proceeding to cleanup", len(oldLogFiles))
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

func (w *rollingWriter) zeroLogger() io.Writer {
	return zerolog.ConsoleWriter{
		Out:        w.fw,
		NoColor:    true,
		TimeFormat: timestampFmt,
	}
}
