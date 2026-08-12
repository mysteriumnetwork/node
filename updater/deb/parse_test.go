/*
 * Copyright (C) 2026 The "MysteriumNetwork/node" Authors.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 */

package deb

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testSHA256 = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

func TestParseAndValidateCandidate(t *testing.T) {
	policy := `myst:
  Installed: 1.36.4+build1+jammy
  Candidate: 1.36.5+build2+jammy
  Version table:
     1.36.5+build2+jammy 500
        500 https://ppa.launchpadcontent.net/mysteriumnetwork/node/ubuntu jammy/main amd64 Packages
 *** 1.36.4+build1+jammy 100
        100 /var/lib/dpkg/status
`
	candidate, err := parseCandidate(policy)
	require.NoError(t, err)
	assert.Equal(t, "1.36.5+build2+jammy", candidate)
	assert.NoError(t, validateCandidateSource(policy, candidate))
}

func TestCandidateRejectsAnotherRepository(t *testing.T) {
	policy := `myst:
  Installed: 1.36.4
  Candidate: 9.9.9
  Version table:
     9.9.9 1000
       1000 https://packages.example.test/ubuntu jammy/main amd64 Packages
`
	candidate, err := parseCandidate(policy)
	require.NoError(t, err)
	assert.Error(t, validateCandidateSource(policy, candidate))
}

func TestValidateRepositoryPolicy(t *testing.T) {
	policy := `Package files:
 100 /var/lib/dpkg/status
     release a=now
 500 https://ppa.mysterium.network/mysteriumnetwork/node/ubuntu jammy/main amd64 Packages
     release v=22.04,o=LP-PPA-mysteriumnetwork-node,a=jammy,n=jammy,l=Mysterium Node,c=main,b=amd64
     origin ppa.mysterium.network
`
	assert.NoError(t, validateRepositoryPolicy(policy))
	assert.Error(t, validateRepositoryPolicy(strings.Replace(policy, RepositoryOrigin, "spoofed-origin", 1)))
}

func TestParsePackageMetadata(t *testing.T) {
	output := `Package: myst
Architecture: amd64
Version: 1.36.5+build2+jammy
Filename: pool/main/m/myst/myst_1.36.5+build2+jammy_amd64.deb
SHA256: ` + testSHA256 + `
Description: Mysterium Node
`
	metadata, err := parsePackageMetadata(output, "1.36.5+build2+jammy", "amd64")
	require.NoError(t, err)
	assert.Equal(t, PackageName, metadata.Package)
	assert.Equal(t, testSHA256, metadata.SHA256)
}

func TestPackageMetadataRejectsUnsafeValues(t *testing.T) {
	base := `Package: myst
Architecture: amd64
Version: 1.36.5+build2+jammy
Filename: %s
SHA256: ` + testSHA256 + "\n"
	for _, filename := range []string{
		"/tmp/myst.deb",
		"pool/main/m/myst/../../../../tmp/myst.deb",
		"pool/main/m/myst/myst.tar.gz",
	} {
		_, err := parsePackageMetadata(strings.Replace(base, "%s", filename, 1), "1.36.5+build2+jammy", "amd64")
		assert.Error(t, err, filename)
	}
}

func TestValidateDownloadURI(t *testing.T) {
	metadata := packageMetadata{
		Version:  "1.36.5+build2+jammy",
		Filename: "pool/main/m/myst/myst_1.36.5+build2+jammy_amd64.deb",
		SHA256:   testSHA256,
	}
	output := `'https://ppa.launchpadcontent.net/mysteriumnetwork/node/ubuntu/pool/main/m/myst/myst_1.36.5%2bbuild2%2bjammy_amd64.deb' myst.deb 1234 SHA256:` + testSHA256
	assert.NoError(t, validateDownloadURI(output, metadata))
	assert.Error(t, validateDownloadURI(strings.Replace(output, testSHA256, strings.Repeat("f", 64), 1), metadata))
	assert.Error(t, validateDownloadURI(strings.Replace(output, "ppa.launchpadcontent.net", "evil.example", 1), metadata))
}

func TestAllowedRepositoryURL(t *testing.T) {
	for _, allowed := range []string{
		"https://ppa.launchpadcontent.net/mysteriumnetwork/node/ubuntu/dists/jammy/main/binary-amd64/Packages",
		"https://ppa.mysterium.network/mysteriumnetwork/node/ubuntu/pool/main/m/myst/myst.deb",
	} {
		assert.True(t, isAllowedRepositoryURL(allowed), allowed)
	}
	for _, rejected := range []string{
		"http://ppa.launchpadcontent.net/mysteriumnetwork/node/ubuntu/pool/myst.deb",
		"https://ppa.launchpadcontent.net/another/node/ubuntu/pool/myst.deb",
		"https://ppa.launchpadcontent.net.evil.example/mysteriumnetwork/node/ubuntu/pool/myst.deb",
		"https://user@ppa.launchpadcontent.net/mysteriumnetwork/node/ubuntu/pool/myst.deb",
	} {
		assert.False(t, isAllowedRepositoryURL(rejected), rejected)
	}
}
