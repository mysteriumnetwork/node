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

package mmn

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/rs/zerolog/log"

	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/requests"
)

// NodeInformationDto contains node information to be sent to MMN
type NodeInformationDto struct {
	// local IP is used to give quick access to WebUI from MMN
	LocalIP         string `json:"local_ip"`
	Identity        string `json:"identity"`
	APIKey          string `json:"api_key"`
	VendorID        string `json:"vendor_id"`
	OS              string `json:"os"`
	Arch            string `json:"arch"`
	NodeVersion     string `json:"node_version"`
	LauncherVersion string `json:"launcher_version"`
	HostOSType      string `json:"host_os_type"`
}

// NewClient returns MMN API client
func NewClient(httpClient *requests.HTTPClient, mmnAddress string, signer identity.SignerFactory) *client {
	return &client{
		httpClient: httpClient,
		mmnAddress: mmnAddress,
		signer:     signer,
	}
}

type client struct {
	httpClient *requests.HTTPClient
	mmnAddress string
	signer     identity.SignerFactory
}

// RegisterNode does an HTTP call to MMN and registers node
func (m *client) RegisterNode(info *NodeInformationDto) error {
	log.Debug().Msgf("Registering node to MMN: %+v", *info)

	id := identity.FromAddress(info.Identity)
	req, err := requests.NewSignedPostRequest(m.mmnAddress, "node", info, m.signer(id))
	if err != nil {
		return err
	}

	return m.httpClient.DoRequest(req)
}

// UpdateBeneficiaryRequest is used when setting a new beneficiary in MMN.
type UpdateBeneficiaryRequest struct {
	Beneficiary string `json:"beneficiary"`
	Identity    string `json:"identity"`
}

// UpdateBeneficiary updates beneficiary in MMN
func (m *client) UpdateBeneficiary(data *UpdateBeneficiaryRequest) error {
	log.Debug().Msgf("Updating beneficiary in MMN: %+v", *data)

	id := identity.FromAddress(data.Identity)
	req, err := requests.NewSignedPutRequest(m.mmnAddress, "node/beneficiary", data, m.signer(id))
	if err != nil {
		return err
	}

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("got a non ok response code: %d", resp.StatusCode)
	}

	var respBody struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &respBody); err != nil {
		return err
	}

	if !respBody.Success {
		return fmt.Errorf("update beneficiary request failed with message: %s", respBody.Message)
	}

	return nil
}

// GetBeneficiary get beneficiary from MMN.
func (m *client) GetBeneficiary(identityStr string) (string, error) {
	id := identity.FromAddress(identityStr)
	req, err := requests.NewSignedGetRequest(m.mmnAddress, "node/beneficiary?identity="+identityStr, m.signer(id))
	if err != nil {
		return "", err
	}
	resp, err := m.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return "", fmt.Errorf("got a non ok response code: %d", resp.StatusCode)
	}

	var benef struct {
		Success     bool   `json:"success"`
		Message     string `json:"message"`
		Beneficiary string `json:"beneficiary"`
	}
	if err := json.Unmarshal(body, &benef); err != nil {
		return "", err
	}

	if !benef.Success {
		if strings.Contains(benef.Message, "not found") {
			return "", nil
		}

		return "", fmt.Errorf("beneficiary get request failed with message: %s", benef.Message)
	}

	return benef.Beneficiary, nil
}

// GetReport does an HTTP call to MMN and fetches node report
func (m *client) GetReport(identityStr string) (string, error) {
	id := identity.FromAddress(identityStr)
	req, err := requests.NewSignedGetRequest(m.mmnAddress, "node/report?identity="+identityStr, m.signer(id))
	if err != nil {
		return "", err
	}

	res, err := m.httpClient.Do(req)
	if err != nil {
		return "", err
	}

	body, err := ioutil.ReadAll(res.Body)

	return string(body), nil
}
