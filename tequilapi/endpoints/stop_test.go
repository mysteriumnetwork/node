/*
 * Copyright (C) 2018 The "MysteriumNetwork/node" Authors.
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

package endpoints

import (
	"context"
	"github.com/julienschmidt/httprouter"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type fakeStopper struct {
	stopAllowed chan struct{}
	stopped     chan struct{}
}

func (fs *fakeStopper) AllowStop() {
	fs.stopAllowed <- struct{}{}
}

func (fs *fakeStopper) Stop() {
	<-fs.stopAllowed
	fs.stopped <- struct{}{}
}

func TestAddRouteForStop(t *testing.T) {
	stopper := fakeStopper{
		stopAllowed: make(chan struct{}, 1),
		stopped:     make(chan struct{}, 1),
	}
	router := httprouter.New()
	AddRouteForStop(router, stopper.Stop)

	resp := httptest.NewRecorder()

	cancelCtx, finishRequestHandling := context.WithCancel(context.Background())
	req := httptest.NewRequest("POST", "/stop", strings.NewReader("")).WithContext(cancelCtx)
	router.ServeHTTP(resp, req)
	assert.Equal(t, http.StatusAccepted, resp.Code)
	assert.Equal(t, 0, len(stopper.stopped))

	stopper.AllowStop()
	finishRequestHandling()

	select {
	case <-stopper.stopped:
	case <-time.After(time.Second):
		t.Error("Stopper was not executed")
	}
}
