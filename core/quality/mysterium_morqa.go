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

package quality

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"

	"github.com/mysteriumnetwork/metrics"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/requests"
)

const (
	mysteriumMorqaAgentName = "goclient-v0.1"

	maxBatchMetricsToKeep = 100
	maxBatchMetricsToWait = 30 * time.Second
)

type metric struct {
	owner string
	event *metrics.Event
}

// MysteriumMORQA HTTP client for Mysterium Quality Oracle - MORQA.
type MysteriumMORQA struct {
	baseURL string
	client  *requests.HTTPClient
	signer  identity.SignerFactory

	batch    metrics.Batch
	eventsMu sync.Mutex
	metrics  chan metric

	once sync.Once
	stop chan struct{}
}

// NewMorqaClient creates Mysterium Morqa client with a real communication.
func NewMorqaClient(httpClient *requests.HTTPClient, baseURL string, signer identity.SignerFactory) *MysteriumMORQA {
	morqa := &MysteriumMORQA{
		baseURL: baseURL,
		client:  httpClient,
		signer:  signer,

		metrics: make(chan metric, maxBatchMetricsToKeep),
		stop:    make(chan struct{}),
	}

	return morqa
}

// Start starts sending batch metrics to the Morqa server.
func (m *MysteriumMORQA) Start() {
	trigger := time.After(maxBatchMetricsToWait)

	for {
		select {
		case metric := <-m.metrics:
			event, err := m.signMetric(metric)
			if err != nil {
				log.Error().Err(err).Msg("Failed to sign metrics event")

				continue
			}

			m.addMetric(event)

			m.eventsMu.Lock()
			size := len(m.batch.Events)
			m.eventsMu.Unlock()

			if size < maxBatchMetricsToKeep {
				continue
			}
		case <-trigger:
		case <-m.stop:
			return
		}

		if err := m.sendMetrics(); err != nil {
			log.Error().Err(err).Msg("Failed to sent batch metrics request")
		}

		trigger = time.After(maxBatchMetricsToWait)
	}
}

// signMetric creates signature and adds to to the metrics event.
func (m *MysteriumMORQA) signMetric(metric metric) (*metrics.Event, error) {
	bin, err := proto.Marshal(metric.event)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metrics event: %w", err)
	}

	signature, err := m.signer(identity.FromAddress(metric.owner)).Sign(bin)
	if err != nil {
		return nil, fmt.Errorf("failed to sing metrics event: %w", err)
	}

	metric.event.Signature = signature.Base64()

	return metric.event, nil
}

// Stop sends the final metrics to the MORQA and stops the sending process.
func (m *MysteriumMORQA) Stop() {
	m.once.Do(func() {
		close(m.stop)
	})

	if err := m.sendMetrics(); err != nil {
		log.Error().Err(err).Msg("Failed to sent batch metrics request on close")
	}
}

func (m *MysteriumMORQA) addMetric(event *metrics.Event) {
	m.eventsMu.Lock()
	defer m.eventsMu.Unlock()

	switch event.Metric.(type) {
	case *metrics.Event_SessionStatisticsPayload: // Allow sending only the last session statistics payload in a single batch.
		for i, e := range m.batch.Events {
			if _, ok := e.Metric.(*metrics.Event_SessionStatisticsPayload); ok {
				m.batch.Events[i] = event
				return
			}
		}
	case *metrics.Event_PingEvent: // Allow sending only the last ping event in a single batch.
		for i, e := range m.batch.Events {
			if _, ok := e.Metric.(*metrics.Event_PingEvent); ok {
				m.batch.Events[i] = event
				return
			}
		}
	}

	m.batch.Events = append(m.batch.Events, event)
}

func (m *MysteriumMORQA) sendMetrics() error {
	m.eventsMu.Lock()
	defer m.eventsMu.Unlock()

	if len(m.batch.Events) == 0 {
		return nil
	}

	request, err := m.newRequestBinary(http.MethodPost, "batch", &m.batch)
	if err != nil {
		return err
	}

	request.Close = true

	response, err := m.client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if err := parseResponseError(response); err != nil {
		return err
	}

	m.batch = metrics.Batch{}

	return nil
}

// ProposalsQuality returns a list of proposals with a quality parameters.
func (m *MysteriumMORQA) ProposalsQuality() []ProposalQuality {
	request, err := m.newRequestJSON(http.MethodGet, "providers/quality", nil)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to create proposals quality request")

		return nil
	}

	response, err := m.client.Do(request)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to request proposals quality")

		return nil
	}
	defer response.Body.Close()

	var qualityResponse []ProposalQuality
	if err = parseResponseJSON(response, &qualityResponse); err != nil {
		log.Warn().Err(err).Msg("Failed to parse proposals quality")

		return nil
	}

	return qualityResponse
}

// ProviderSessions fetch provider sessions from prometheus
func (m *MysteriumMORQA) ProviderSessions(providerID string) []ProviderSession {
	request, err := m.newRequestJSON(http.MethodGet, fmt.Sprintf("providers/sessions?provider_id=%s", providerID), nil)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to create proposals quality request")

		return nil
	}

	response, err := m.client.Do(request)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to request proposals quality")

		return nil
	}
	defer response.Body.Close()

	var responseBody struct {
		Connects []ProviderSession `json:"connects"`
	}
	if err = parseResponseJSON(response, &responseBody); err != nil {
		log.Warn().Err(err).Msg("Failed to parse proposals quality")

		return nil
	}
	return responseBody.Connects
}

// SendMetric submits new metric.
func (m *MysteriumMORQA) SendMetric(id string, event *metrics.Event) error {
	m.metrics <- metric{
		owner: id,
		event: event,
	}

	return nil
}

func (m *MysteriumMORQA) newRequest(method, path string, body []byte) (*http.Request, error) {
	url := m.baseURL
	if len(path) > 0 {
		url = fmt.Sprintf("%v/%v", url, path)
	}

	request, err := http.NewRequest(method, url, bytes.NewBuffer(body))
	request.Header.Set("User-Agent", mysteriumMorqaAgentName)
	request.Header.Set("Accept", "application/json")

	return request, err
}

func (m *MysteriumMORQA) newRequestJSON(method, path string, payload interface{}) (*http.Request, error) {
	payloadBody, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := m.newRequest(method, path, payloadBody)
	req.Header.Set("Accept", "application/json")

	return req, err
}

func (m *MysteriumMORQA) newRequestBinary(method, path string, payload proto.Message) (*http.Request, error) {
	payloadBody, err := proto.Marshal(payload)
	if err != nil {
		return nil, err
	}

	request, err := m.newRequest(method, path, payloadBody)
	request.Header.Set("Content-Type", "application/octet-stream")

	return request, err
}

func parseResponseJSON(response *http.Response, dto interface{}) error {
	responseJSON, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}

	return json.Unmarshal(responseJSON, dto)
}

type errorDTO struct {
	Message string `json:"message"`
}

// Sometimes we can get json message with single "message" field which represents error - try to get that.
func parseResponseError(response *http.Response) error {
	if response.StatusCode >= 200 && response.StatusCode < 300 {
		return nil
	}

	var parsedBody errorDTO

	var message string

	err := parseResponseJSON(response, &parsedBody)
	if err != nil {
		message = err.Error()
	} else {
		message = parsedBody.Message
	}

	return fmt.Errorf("server response invalid: %s (%s). Possible error: %s", response.Status, response.Request.URL, message)
}
