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

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"

	"github.com/mysteriumnetwork/metrics"

	"github.com/mysteriumnetwork/node/core/node"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/requests"
)

const (
	mysteriumMorqaAgentName = "goclient-v0.1"

	maxBatchMetricsToKeep = 100
	maxBatchMetricsToWait = 30 * time.Second

	maxBatchSentFails = 3
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

	batch    map[string]*batchWithTimeout
	eventsMu sync.RWMutex
	metrics  chan metric

	once sync.Once
	stop chan struct{}
}

type batchWithTimeout struct {
	batch    *metrics.Batch
	failed   int
	lastFail time.Time
}

// NewMorqaClient creates Mysterium Morqa client with a real communication.
func NewMorqaClient(httpClient *requests.HTTPClient, baseURL string, signer identity.SignerFactory) *MysteriumMORQA {
	morqa := &MysteriumMORQA{
		baseURL: baseURL,
		client:  httpClient,
		signer:  signer,

		batch:   make(map[string]*batchWithTimeout),
		metrics: make(chan metric, 1000*maxBatchMetricsToKeep),
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
			m.addMetric(metric)

			m.eventsMu.RLock()
			size := len(m.batch[metric.owner].batch.Events)
			failed := m.batch[metric.owner].failed
			lastFailed := m.batch[metric.owner].lastFail
			m.eventsMu.RUnlock()

			if size < maxBatchMetricsToKeep {
				continue
			}

			if failed > maxBatchSentFails {
				m.eventsMu.Lock()
				delete(m.batch, metric.owner)
				m.eventsMu.Unlock()
			}

			if time.Now().Before(lastFailed.Add(maxBatchMetricsToWait)) {
				continue
			}

		case <-trigger:
		case <-m.stop:
			return
		}

		m.sendAll()

		trigger = time.After(maxBatchMetricsToWait)
	}
}

// signBatch creates signature for the metrics batch.
func (m *MysteriumMORQA) signBatch(owner string, batch *metrics.Batch) (string, error) {
	bin, err := proto.Marshal(batch)
	if err != nil {
		return "", fmt.Errorf("failed to marshal metrics event: %w", err)
	}

	signature, err := m.signer(identity.FromAddress(owner)).Sign(bin)
	if err != nil {
		return "", fmt.Errorf("failed to sing metrics event: %w", err)
	}

	return signature.Base64(), nil
}

// Stop sends the final metrics to the MORQA and stops the sending process.
func (m *MysteriumMORQA) Stop() {
	m.once.Do(func() {
		close(m.stop)
	})

	m.sendAll()
}

func (m *MysteriumMORQA) addMetric(metric metric) {
	m.eventsMu.Lock()
	defer m.eventsMu.Unlock()

	batch, ok := m.batch[metric.owner]
	if !ok || batch == nil {
		batch = &batchWithTimeout{
			batch: &metrics.Batch{},
		}
		m.batch[metric.owner] = batch
	}

	switch metric.event.Metric.(type) {
	case *metrics.Event_SessionStatisticsPayload: // Allow sending only the last session statistics payload in a single batch.
		for i, e := range m.batch[metric.owner].batch.Events {
			if _, ok := e.Metric.(*metrics.Event_SessionStatisticsPayload); ok {
				m.batch[metric.owner].batch.Events[i] = metric.event
				return
			}
		}
	case *metrics.Event_PingEvent: // Allow sending only the last ping event in a single batch.
		for i, e := range m.batch[metric.owner].batch.Events {
			if _, ok := e.Metric.(*metrics.Event_PingEvent); ok {
				m.batch[metric.owner].batch.Events[i] = metric.event
				return
			}
		}
	}

	batch.batch.Events = append(batch.batch.Events, metric.event)
	m.batch[metric.owner] = batch
}

func (m *MysteriumMORQA) sendAll() {
	m.eventsMu.Lock()
	defer m.eventsMu.Unlock()

	for owner := range m.batch {
		err := m.sendMetrics(owner)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to sent batch metrics request, %v", len(m.batch[owner].batch.GetEvents()))
			m.batch[owner].failed++
			m.batch[owner].lastFail = time.Now()
		}
	}
}

func (m *MysteriumMORQA) sendMetrics(owner string) error {
	if len(m.batch[owner].batch.Events) == 0 {
		return nil
	}

	batch := m.batch[owner].batch

	signature, err := m.signBatch(owner, batch)
	if err != nil {
		log.Error().Err(err).Msg("Failed to sign metrics event")
	}

	sb := &metrics.SignedBatch{
		Signature: signature,
		Batch:     batch,
	}

	request, err := m.newRequestBinary(http.MethodPost, "batch", sb)
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

	delete(m.batch, owner)

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

// ProviderStatuses fetch provider connectivity statuses from quality oracle.
func (m *MysteriumMORQA) ProviderStatuses(providerID string) (node.MonitoringAgentStatuses, error) {
	id := identity.FromAddress(providerID)

	request, err := requests.NewSignedGetRequest(m.baseURL, "provider/statuses", m.signer(id))
	if err != nil {
		return nil, err
	}

	response, err := m.client.Do(request)
	if err != nil {
		log.Err(err).Msg("Failed to request provider monitoring agent statuses")
		return nil, err
	}
	defer response.Body.Close()

	var statuses node.MonitoringAgentStatuses

	if err = parseResponseJSON(response, &statuses); err != nil {
		log.Err(err).Msg("Failed to parse provider monitoring agent statuses")
		return nil, err
	}

	return statuses, nil
}

// ProviderSessionsList fetch provider sessions list from quality oracle.
func (m *MysteriumMORQA) ProviderSessionsList(id identity.Identity, rangeTime string) ([]node.SessionItem, error) {
	request, err := requests.NewSignedGetRequest(m.baseURL, fmt.Sprintf("provider/sessions?range=%s", rangeTime), m.signer(id))
	if err != nil {
		return nil, err
	}

	response, err := m.client.Do(request)
	if err != nil {
		return nil, errors.Wrap(err, "failed to request provider monitoring sessions list")
	}
	defer response.Body.Close()

	var sessions []node.SessionItem

	if err = parseResponseJSON(response, &sessions); err != nil {
		log.Err(err).Msg("Failed to parse provider monitoring sessions list")
		return nil, err
	}

	return sessions, nil
}

// ProviderTransferredData fetch total traffic served by the provider during a period of time from quality oracle.
func (m *MysteriumMORQA) ProviderTransferredData(id identity.Identity, rangeTime string) (node.TransferredData, error) {
	var data node.TransferredData
	request, err := requests.NewSignedGetRequest(m.baseURL, fmt.Sprintf("provider/transferred-data?range=%s", rangeTime), m.signer(id))
	if err != nil {
		return data, err
	}

	response, err := m.client.Do(request)
	if err != nil {
		return data, errors.Wrap(err, "failed to request provider transferred data")
	}
	defer response.Body.Close()

	if err = parseResponseJSON(response, &data); err != nil {
		log.Err(err).Msg("Failed to parse provider transferred data")
		return data, err
	}

	return data, nil
}

// ProviderSessionsCount fetch provider sessions number from quality oracle.
func (m *MysteriumMORQA) ProviderSessionsCount(id identity.Identity, rangeTime string) (node.SessionsCount, error) {
	var count node.SessionsCount
	request, err := requests.NewSignedGetRequest(m.baseURL, fmt.Sprintf("provider/sessions-count?range=%s", rangeTime), m.signer(id))
	if err != nil {
		return count, err
	}

	response, err := m.client.Do(request)
	if err != nil {
		return count, errors.Wrap(err, "failed to request provider monitoring sessions count")
	}
	defer response.Body.Close()

	if err = parseResponseJSON(response, &count); err != nil {
		log.Err(err).Msg("Failed to parse provider monitoring sessions count")
		return count, err
	}

	return count, nil
}

// ProviderConsumersCount fetch consumers number served by provider from quality oracle.
func (m *MysteriumMORQA) ProviderConsumersCount(id identity.Identity, rangeTime string) (node.ConsumersCount, error) {
	var count node.ConsumersCount
	request, err := requests.NewSignedGetRequest(m.baseURL, fmt.Sprintf("provider/consumers-count?range=%s", rangeTime), m.signer(id))
	if err != nil {
		return count, err
	}

	response, err := m.client.Do(request)
	if err != nil {
		return count, errors.Wrap(err, "failed to request provider monitoring consumers count")
	}
	defer response.Body.Close()

	if err = parseResponseJSON(response, &count); err != nil {
		log.Err(err).Msg("Failed to parse provider monitoring consumers count")
		return count, err
	}

	return count, nil
}

// ProviderEarningsSeries fetch earnings data series metrics from quality oracle.
func (m *MysteriumMORQA) ProviderEarningsSeries(id identity.Identity, rangeTime string) (node.EarningsSeries, error) {
	var data node.EarningsSeries
	request, err := requests.NewSignedGetRequest(m.baseURL, fmt.Sprintf("provider/series-earnings?range=%s", rangeTime), m.signer(id))
	if err != nil {
		return data, err
	}

	response, err := m.client.Do(request)
	if err != nil {
		return data, errors.Wrap(err, "failed to request provider series earnings")
	}
	defer response.Body.Close()

	if err = parseResponseJSON(response, &data); err != nil {
		log.Err(err).Msg("Failed to parse provider series earnings")
		return data, err
	}

	return data, nil
}

// ProviderSessionsSeries fetch earnings data series metrics from quality oracle.
func (m *MysteriumMORQA) ProviderSessionsSeries(id identity.Identity, rangeTime string) (node.SessionsSeries, error) {
	var data node.SessionsSeries
	request, err := requests.NewSignedGetRequest(m.baseURL, fmt.Sprintf("provider/series-sessions?range=%s", rangeTime), m.signer(id))
	if err != nil {
		return data, err
	}

	response, err := m.client.Do(request)
	if err != nil {
		return data, errors.Wrap(err, "failed to request provider series sessions")
	}
	defer response.Body.Close()

	if err = parseResponseJSON(response, &data); err != nil {
		log.Err(err).Msg("Failed to parse provider series sessions")
		return data, err
	}

	return data, nil
}

// ProviderTransferredDataSeries fetch transferred bytes data series metrics from quality oracle.
func (m *MysteriumMORQA) ProviderTransferredDataSeries(id identity.Identity, rangeTime string) (node.TransferredDataSeries, error) {
	var data node.TransferredDataSeries
	request, err := requests.NewSignedGetRequest(m.baseURL, fmt.Sprintf("provider/series-data?range=%s", rangeTime), m.signer(id))
	if err != nil {
		return data, err
	}

	response, err := m.client.Do(request)
	if err != nil {
		return data, errors.Wrap(err, "failed to request provider series data")
	}
	defer response.Body.Close()

	if err = parseResponseJSON(response, &data); err != nil {
		log.Err(err).Msg("Failed to parse provider series data")
		return data, err
	}

	return data, nil
}

// ProviderActivityStats fetch provider activity stats from quality oracle.
func (m *MysteriumMORQA) ProviderActivityStats(id identity.Identity) (node.ActivityStats, error) {
	var stats node.ActivityStats
	request, err := requests.NewSignedGetRequest(m.baseURL, fmt.Sprintf("provider/activity-stats"), m.signer(id))
	if err != nil {
		return stats, err
	}

	response, err := m.client.Do(request)
	if err != nil {
		return stats, errors.Wrap(err, "failed to request provider activity stats")
	}
	defer response.Body.Close()

	if err = parseResponseJSON(response, &stats); err != nil {
		log.Err(err).Msg("Failed to parse provider activity stats")
		return stats, err
	}

	return stats, nil
}

// ProviderQuality fetch provider quality from quality oracle.
func (m *MysteriumMORQA) ProviderQuality(id identity.Identity) (node.QualityInfo, error) {
	var res node.QualityInfo
	request, err := requests.NewSignedGetRequest(m.baseURL, "provider/quality", m.signer(id))
	if err != nil {
		return res, err
	}

	response, err := m.client.Do(request)
	if err != nil {
		return res, errors.Wrap(err, "failed to request provider quality")
	}
	defer response.Body.Close()

	if err = parseResponseJSON(response, &res); err != nil {
		log.Err(err).Msg("Failed to parse provider quality")
		return res, err
	}

	return res, nil
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
