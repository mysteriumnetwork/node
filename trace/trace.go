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
	"sort"
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
func NewTracer() *Tracer {
	return &Tracer{
		stages: make(map[string]*stage),
	}
}

// Tracer represents tracer which records stages durations. It can be used
// to record stages times for tracing how long it took time.
type Tracer struct {
	mu       sync.Mutex
	stages   map[string]*stage
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

	if _, ok := t.stages[key]; ok {
		log.Error().Msgf("Stage %s was already started", key)
		return ""
	}

	t.stages[key] = &stage{
		key:   key,
		start: time.Now(),
	}
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

	if s, ok := t.stages[key]; ok {
		s.end = time.Now()
	} else {
		log.Error().Msgf("Stage %s was not started", key)
	}
}

// Finish finishes tracing and returns formatted string with stages durations.
func (t *Tracer) Finish(eventPublisher eventbus.Publisher, id string) string {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.finished = true

	// Sort stages by start time.
	var stages []*stage
	for _, v := range t.stages {
		stages = append(stages, v)
	}
	sort.Slice(stages, func(i, j int) bool {
		return stages[i].start.Before(stages[j].start)
	})

	var strs []string
	for _, s := range stages {
		t.publishStageEvent(eventPublisher, id, *s)
		if s.end.After(time.Time{}) {
			strs = append(strs, fmt.Sprintf("%q took %s", s.key, s.end.Sub(s.start).String()))
		} else {
			strs = append(strs, fmt.Sprintf("%q did not start", s.key))
		}
	}

	return strings.Join(strs, ", ")
}

func (t *Tracer) publishStageEvent(eventPublisher eventbus.Publisher, id string, stage stage) {
	if eventPublisher == nil {
		return
	}

	go eventPublisher.Publish(AppTopicTraceEvent,
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
