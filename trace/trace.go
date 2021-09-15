/*
 * Copyright (C) 2020 The "MysteriumNetwork/node" Authors.
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

package trace

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/mysteriumnetwork/node/eventbus"
	"github.com/rs/zerolog/log"
)

const (
	// AppTopicTraceEvent represents event topic for Trace events
	AppTopicTraceEvent = "Trace"
)

// NewTracer returns new tracer instance.
func NewTracer(name string) *Tracer {
	tracer := &Tracer{
		stages: make([]*stage, 0),
	}
	tracer.name = tracer.StartStage(name)
	return tracer
}

// Tracer represents tracer which records stages durations. It can be used
// to record stages times for tracing how long it took time.
type Tracer struct {
	name     string
	mu       sync.Mutex
	stages   []*stage
	finished bool
}

// StartStage starts tracing stage for given key.
func (t *Tracer) StartStage(key string) string {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.finished {
		log.Error().Msg("Tracer is already finished")
		return ""
	}
	if _, ok := t.findStage(key); ok {
		log.Error().Msgf("Stage %s was already started", key)
		return ""
	}

	t.stages = append(t.stages, &stage{
		key:   key,
		start: time.Now(),
	})
	return key
}

// EndStage ends tracing stage for given key.
func (t *Tracer) EndStage(key string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.finished {
		log.Error().Msg("Tracer is already finished")
		return
	}
	s, ok := t.findStage(key)
	if !ok {
		log.Error().Msgf("Stage %s was not started", key)
		return
	}

	s.end = time.Now()
}

// Finish finishes tracing and returns formatted string with stages durations.
func (t *Tracer) Finish(eventPublisher eventbus.Publisher, id string) string {
	t.EndStage(t.name)

	t.mu.Lock()
	defer t.mu.Unlock()
	t.finished = true

	var strs []string
	for _, s := range t.stages {
		if s.end.After(time.Time{}) {
			t.publishStageEvent(eventPublisher, id, *s)
			strs = append(strs, fmt.Sprintf("%q took %s", s.key, s.end.Sub(s.start).String()))
		} else {
			strs = append(strs, fmt.Sprintf("%q did not start", s.key))
		}
	}

	return strings.Join(strs, ", ")
}

func (t *Tracer) findStage(key string) (*stage, bool) {
	for _, s := range t.stages {
		if s.key == key {
			return s, true
		}
	}
	return nil, false
}

func (t *Tracer) publishStageEvent(eventPublisher eventbus.Publisher, id string, stage stage) {
	if eventPublisher == nil {
		return
	}

	eventPublisher.Publish(AppTopicTraceEvent,
		Event{
			ID:       id,
			Key:      stage.key,
			Duration: stage.end.Sub(stage.start),
		},
	)
}

type stage struct {
	key        string
	start, end time.Time
}

// Event represents a published Trace event.
type Event struct {
	ID       string
	Key      string
	Duration time.Duration
}
