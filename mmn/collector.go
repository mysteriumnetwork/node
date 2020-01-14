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
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"

	"github.com/mysteriumnetwork/node/config"
	"github.com/mysteriumnetwork/node/core/ip"
	"github.com/mysteriumnetwork/node/metadata"
)

// Collector collects environment data
type Collector struct {
	ipResolver ip.Resolver
	node       *NodeInformationDto
}

// NewCollector creates new environment data collector struct
func NewCollector(resolver ip.Resolver) *Collector {
	return &Collector{ipResolver: resolver}
}

// CollectEnvironmentInformation sends node information to MMN on identity unlock
func (c *Collector) CollectEnvironmentInformation() error {
	outboundIp, err := c.ipResolver.GetOutboundIPAsString()
	if err != nil {
		return errors.Wrap(err, "Failed to get Outbound IP")
	}

	mac, err := ip.GetMACAddressForIP(outboundIp)
	if err != nil {
		return errors.Wrap(err, "Failed to get MAC address")
	}

	node := &NodeInformationDto{
		Arch:        runtime.GOOS + "/" + runtime.GOARCH,
		OS:          getOS(),
		NodeVersion: metadata.VersionAsString(),
		MACAddress:  hashMACAddress(mac),
		LocalIP:     outboundIp,
		VendorID:    config.GetString(config.FlagVendorID),
		IsProvider:  false,
		IsClient:    false,
	}

	c.node = node

	return nil
}

// SetIdentity sets node's identity
func (c *Collector) SetIdentity(identity string) {
	c.node.Identity = identity
}

// SetIsProvider marks node as a provider node
func (c *Collector) SetIsProvider(isProvider bool) {
	c.node.IsProvider = isProvider
}

// SetIsClient marks node as a client node
func (c *Collector) SetIsClient(isClient bool) {
	c.node.IsClient = isClient
}

// IsClient returns if the node is a client node
func (c *Collector) IsClient() bool {
	return c.node.IsClient
}

// IsProvider returns if the node is a provider node
func (c *Collector) IsProvider() bool {
	return c.node.IsProvider
}

// GetCollectedInformation returns collected information
func (c *Collector) GetCollectedInformation() *NodeInformationDto {
	return c.node
}

func hashMACAddress(mac string) string {
	sha256Bytes := sha256.Sum256([]byte(mac))

	return hex.EncodeToString(sha256Bytes[:])
}

func getOS() string {
	switch runtime.GOOS {
	case "darwin":
		output, err := exec.Command("sw_vers", "-productVersion").Output()
		if err != nil {
			log.Err(err).Msg("Failed to get OS information")
			return "macOS (unknown)"
		}
		return "macOS " + strings.TrimSpace(string(output))
	case "linux":
		distro, err := parseLinuxOS()
		if err != nil {
			log.Err(err).Msg("Failed to get OS information")
			return "linux (unknown)"
		}
		return distro
	case "windows":
		output, err := exec.Command("wmic", "os", "get", "Caption", "/value").Output()
		if err != nil {
			log.Err(err).Msg("Failed to get OS information")
			return "windows (unknown)"
		}
		return strings.TrimSpace(strings.TrimPrefix(string(output), "Caption="))
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
