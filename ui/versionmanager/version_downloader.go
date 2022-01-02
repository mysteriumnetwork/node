/*
 * Copyright (C) 2021 The "MysteriumNetwork/node" Authors.
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

package versionmanager

import (
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

type dlStatus string

const (
	inProgress dlStatus = "in_progress"
	failed     dlStatus = "failed"
	idle       dlStatus = "idle"
	done       dlStatus = "done"
)

// Downloader node UI downloader
type Downloader struct {
	http *http.Client
	lock sync.Mutex

	progressUpdateEvery time.Duration

	status     Status
	statusLock sync.Mutex
}

// NewDownloader constructor for Downloader
func NewDownloader() *Downloader {
	return &Downloader{
		http: &http.Client{
			Timeout: time.Minute * 5,
		},
		status:              Status{Status: idle},
		progressUpdateEvery: time.Second,
	}
}

// DownloadOpts download options
type DownloadOpts struct {
	URL      *url.URL
	DistFile string
	Tag      string
	Callback func(opts DownloadOpts) error
}

// DownloadNodeUI download node UI
func (d *Downloader) DownloadNodeUI(opts DownloadOpts) {
	d.lock.Lock()
	defer d.lock.Unlock()

	if d.status.Status == inProgress {
		log.Warn().Msg("node UI download in progress - skipping")
		return
	}

	d.update(Status{Status: inProgress, Tag: opts.Tag})

	go d.download(opts)
}

// Status download status
// swagger:model DownloadStatus
type Status struct {
	Status      dlStatus `json:"status"`
	ProgressPct int      `json:"progress_percent"`
	Tag         string   `json:"tag,omitempty"`
	Err         error    `json:"error,omitempty"`
}

func (s Status) withPct(p int) Status {
	s.ProgressPct = p
	return s
}

func (s Status) transition(status dlStatus) Status {
	s.Status = status
	return s
}

func (s Status) transitionWithErr(status dlStatus, err error) Status {
	s.Status = status
	s.Err = err
	return s
}

// Status current download status
func (d *Downloader) Status() Status {
	return d.status
}

func (d *Downloader) download(opts DownloadOpts) {
	head, err := d.http.Head(opts.URL.String())
	if err != nil {
		d.update(d.status.transitionWithErr(failed, err))
		return
	}

	totalSize, err := strconv.Atoi(head.Header.Get("Content-Length"))
	if err != nil {
		d.update(d.status.transitionWithErr(failed, err))
		return
	}

	outFile, err := os.Create(opts.DistFile)
	if err != nil {
		d.update(d.status.transitionWithErr(failed, err))
		return
	}

	doneChan := make(chan struct{})
	go d.progressUpdater(doneChan, outFile, totalSize)

	res, err := d.http.Get(opts.URL.String())
	if err != nil {
		d.update(d.status.transitionWithErr(failed, err))
		return
	}
	defer res.Body.Close()

	_, err = io.Copy(outFile, res.Body)
	if err != nil {
		d.update(d.status.transitionWithErr(failed, err))
		return
	}

	close(doneChan)

	if opts.Callback != nil && d.status.Status != failed {
		log.Info().Msg("executing post download callback")
		err := opts.Callback(opts)
		if err != nil {
			d.update(d.status.transitionWithErr(failed, err))
			return
		}
	} else {
		log.Warn().Msgf("download status is %s - skipping callback", d.status.Status)
	}

	d.update(d.status.transition(done))
}

func (d *Downloader) progressUpdater(done chan struct{}, file *os.File, totalSize int) {
	for {
		select {
		case <-done:
			d.update(d.status.withPct(100))
			return
		case <-time.After(d.progressUpdateEvery):
			f, err := file.Stat()
			if err != nil {
				log.Error().Err(err).Msgf("failed to file.Stat()")
				return
			}

			p := float64(f.Size()) / float64(totalSize) * 100
			d.update(d.status.withPct(int(p)))
		}
	}
}

func (d *Downloader) update(s Status) {
	d.statusLock.Lock()
	defer d.statusLock.Unlock()
	d.status = s

	if s.Err != nil {
		log.Error().Err(s.Err).Msgf("node UI download transitioned to %+v", s)
	} else {
		log.Info().Msgf("node UI download transitioned to %+v", s)
	}
}
