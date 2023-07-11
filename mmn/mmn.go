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
	"bufio"
	"encoding/base64"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/mysteriumnetwork/node/tequilapi/pkce"

	"github.com/mysteriumnetwork/node/tequilapi/sso"

	"github.com/rs/zerolog/log"

	"github.com/mysteriumnetwork/node/config"
	"github.com/mysteriumnetwork/node/core/ip"
	nodevent "github.com/mysteriumnetwork/node/core/node/event"
	"github.com/mysteriumnetwork/node/core/service/servicestate"
	"github.com/mysteriumnetwork/node/eventbus"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/metadata"
)

// MMN struct
type MMN struct {
	client     *client
	ipResolver ip.Resolver

	lastIP       string
	lastIdentity identity.Identity

	mystnodesURL   string
	claimPath      string
	onboardingPath string
}

// NewMMN creates new instance of MMN
func NewMMN(resolver ip.Resolver, client *client) *MMN {
	return &MMN{client: client, ipResolver: resolver, mystnodesURL: config.GetString(config.FlagMMNAddress), claimPath: "/node-claim", onboardingPath: "/clickboarding"}
}

// Subscribe subscribes to node events and reports them to MMN
func (m *MMN) Subscribe(eventBus eventbus.EventBus) error {
	if err := eventBus.SubscribeAsync(nodevent.AppTopicNode, m.handleNodeStart); err != nil {
		return err
	}
	if err := eventBus.SubscribeAsync(identity.AppTopicIdentityUnlock, m.handleIdentityUnlock); err != nil {
		return err
	}
	return eventBus.SubscribeAsync(servicestate.AppTopicServiceStatus, m.handleServiceStart)
}

// handleNodeStart handles node state change and fetches the IP accordingly.
func (m *MMN) handleNodeStart(e nodevent.Payload) {
	if e.Status != nodevent.StatusStarted {
		return
	}

	var err error
	m.lastIP, err = m.ipResolver.GetOutboundIP()
	if err != nil {
		log.Error().Msgf("Failed to get get Outbound IP for MMN: %v", err)
	}
}

func (m *MMN) handleIdentityUnlock(ev identity.AppEventIdentityUnlock) {
	m.lastIdentity = ev.ID
}

// handleServiceStart does auto-register to MMN, but only for providers.
func (m *MMN) handleServiceStart(e servicestate.AppEventServiceStatus) {
	if e.Status != string(servicestate.Running) {
		return
	}

	// TODO Turn off auto-register then WEB UI will have possibility to configure API key
	isRegistrationEnabled := len(config.Current.GetString(config.FlagMMNAPIKey.Name)) != 0
	if !isRegistrationEnabled {
		log.Debug().Msg("Identity unlocked, registration to MMN disabled because the API key missing in config.")
		return
	}

	if err := m.ClaimNode(); err != nil {
		log.Error().Msgf("Failed to register identity to MMN: %v", err)
	}
}

func (m *MMN) claimRequestNoRedirect() NodeClaimRequest {
	return m.claimRequest(nil)
}

func (m *MMN) claimRequest(redirectURL *url.URL) NodeClaimRequest {
	rru := ""
	if redirectURL != nil {
		rru = fmt.Sprint(redirectURL)
	}
	return NodeClaimRequest{
		LocalIP:     m.lastIP,
		Identity:    m.lastIdentity.Address,
		APIKey:      config.GetString(config.FlagMMNAPIKey),
		VendorID:    config.GetString(config.FlagVendorID),
		Arch:        runtime.GOOS + docker() + "/" + runtime.GOARCH,
		OS:          getOS(),
		NodeVersion: metadata.VersionAsString(),
		RedirectURL: rru,
	}
}

func (m *MMN) onboardingRequest(info pkce.Info, redirectURL *url.URL) sso.MystnodesMessage {
	return sso.MystnodesMessage{
		CodeChallenge: info.CodeChallenge,
		Identity:      m.lastIdentity.Address,
		RedirectURL:   fmt.Sprint(redirectURL),
	}
}

// ClaimNode registers node to MMN
func (m *MMN) ClaimNode() error {
	return m.client.ClaimNode(m.claimRequestNoRedirect())
}

// ClaimLink generate claim link
func (m *MMN) ClaimLink(redirectURL *url.URL) (*url.URL, error) {
	claimRequestJson, err := m.claimRequest(redirectURL).json()
	if err != nil {
		return nil, err
	}

	signature, err := m.client.signer(m.lastIdentity).Sign(claimRequestJson)
	if err != nil {
		return nil, err
	}

	link, err := url.Parse(m.mystnodesURL)
	if err != nil {
		return nil, err
	}

	link = link.JoinPath(m.claimPath)

	q := link.Query()
	q.Set("message", base64.RawURLEncoding.EncodeToString(claimRequestJson))
	q.Set("signature", base64.RawURLEncoding.EncodeToString(signature.Bytes()))
	link.RawQuery = q.Encode()

	return link, nil
}

func getOS() string {
	switch runtime.GOOS {
	case "darwin":
		output, err := exec.Command("sw_vers", "-productVersion").Output()
		if err != nil {
			log.Error().Err(err).Msg("Failed to get OS information")
			return "macOS (unknown)"
		}
		return "macOS " + strings.TrimSpace(string(output))
	case "linux":
		distro, err := parseLinuxOS()
		if err != nil {
			log.Error().Err(err).Msg("Failed to get OS information")
			return "linux (unknown)" + docker()
		}
		return distro + docker()
	case "windows":
		output, err := exec.Command("wmic", "os", "get", "Caption", "/value").Output()
		if err != nil {
			log.Error().Err(err).Msg("Failed to get OS information")
			return "windows (unknown)"
		}
		return strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(string(output)), "Caption="))
	}
	return runtime.GOOS
}

func parseLinuxOS() (string, error) {
	output, err := exec.Command("lsb_release", "-d").Output()
	if err == nil {
		return strings.TrimSpace(strings.TrimPrefix(string(output), "Description:")), nil
	}

	const etcOsRelease = "/etc/os-release"
	const altOsRelease = "/usr/lib/os-release"
	osReleaseFile, err := os.Open(etcOsRelease)
	if err != nil {
		if !os.IsNotExist(err) {
			return "", fmt.Errorf("error opening %s: %w", etcOsRelease, err)
		}
		osReleaseFile, err = os.Open(altOsRelease)
		if err != nil {
			return "", fmt.Errorf("error opening %s: %w", altOsRelease, err)
		}
	}
	defer osReleaseFile.Close()

	var prettyName string
	scanner := bufio.NewScanner(osReleaseFile)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "PRETTY_NAME") {
			tokens := strings.SplitN(line, "=", 2)
			if len(tokens) == 2 {
				prettyName = strings.Trim(tokens[1], "\"")
			}
		}
	}
	if prettyName != "" {
		return prettyName, nil
	}

	return "linux (unknown)", nil
}

func docker() string {
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return "(docker)"
	}

	return ""
}
