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
	"bufio"
	"fmt"
	"net/url"
	"path"
	"regexp"
	"strconv"
	"strings"
)

var (
	debianVersionPattern = regexp.MustCompile(`^[0-9A-Za-z.+:~_-]{1,255}$`)
	sha256Pattern        = regexp.MustCompile(`^[0-9a-fA-F]{64}$`)
)

type packageMetadata struct {
	Package      string
	Version      string
	Architecture string
	Filename     string
	SHA256       string
}

func parseCandidate(policy string) (string, error) {
	for _, line := range strings.Split(policy, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "Candidate:") {
			continue
		}
		candidate := strings.TrimSpace(strings.TrimPrefix(line, "Candidate:"))
		if candidate == "" || candidate == "(none)" {
			return "", fmt.Errorf("APT has no candidate for package %s", packageName)
		}
		if !debianVersionPattern.MatchString(candidate) {
			return "", fmt.Errorf("APT returned invalid candidate version %q", candidate)
		}
		return candidate, nil
	}
	return "", fmt.Errorf("APT policy did not contain a candidate")
}

func validateCandidateSource(policy, candidate string) error {
	inCandidate := false
	for _, rawLine := range strings.Split(policy, "\n") {
		line := strings.TrimSpace(rawLine)
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[0] == "***" {
			fields = fields[1:]
		}
		if len(fields) >= 2 {
			if _, err := strconv.Atoi(fields[1]); err == nil && debianVersionPattern.MatchString(fields[0]) {
				inCandidate = fields[0] == candidate
				continue
			}
		}
		if !inCandidate || len(fields) < 2 {
			continue
		}
		if _, err := strconv.Atoi(fields[0]); err != nil {
			continue
		}
		if isAllowedRepositoryURL(fields[1]) {
			return nil
		}
	}
	return fmt.Errorf("candidate %s is not supplied by the Mysterium node PPA", candidate)
}

func validateRepositoryPolicy(policy string) error {
	lines := strings.Split(policy, "\n")
	for index, rawLine := range lines {
		fields := strings.Fields(strings.TrimSpace(rawLine))
		if len(fields) < 2 || !isAllowedRepositoryURL(fields[1]) {
			continue
		}
		end := index + 4
		if end > len(lines) {
			end = len(lines)
		}
		for _, detail := range lines[index+1 : end] {
			if strings.Contains(detail, "o="+repositoryOrigin) {
				return nil
			}
		}
	}
	return fmt.Errorf("authenticated APT metadata for origin %s was not found", repositoryOrigin)
}

func parsePackageMetadata(output, candidate, architecture string) (packageMetadata, error) {
	for _, stanza := range splitParagraphs(output) {
		values := make(map[string]string)
		scanner := bufio.NewScanner(strings.NewReader(stanza))
		for scanner.Scan() {
			key, value, ok := strings.Cut(scanner.Text(), ":")
			if ok {
				values[strings.TrimSpace(key)] = strings.TrimSpace(value)
			}
		}
		if values["Package"] != packageName || values["Version"] != candidate {
			continue
		}
		metadata := packageMetadata{
			Package:      values["Package"],
			Version:      values["Version"],
			Architecture: values["Architecture"],
			Filename:     values["Filename"],
			SHA256:       strings.ToLower(values["SHA256"]),
		}
		if metadata.Architecture != architecture && metadata.Architecture != "all" {
			return packageMetadata{}, fmt.Errorf("candidate architecture %q does not match host %q", metadata.Architecture, architecture)
		}
		if !validPackageFilename(metadata.Filename) {
			return packageMetadata{}, fmt.Errorf("candidate has unsafe package filename %q", metadata.Filename)
		}
		if !sha256Pattern.MatchString(metadata.SHA256) {
			return packageMetadata{}, fmt.Errorf("candidate has invalid SHA256 %q", metadata.SHA256)
		}
		return metadata, nil
	}
	return packageMetadata{}, fmt.Errorf("APT metadata for exact candidate %s was not found", candidate)
}

func validateDownloadURI(output string, metadata packageMetadata) error {
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		rawURI := strings.Trim(fields[0], "'")
		if !isAllowedRepositoryURL(rawURI) {
			continue
		}
		uri, err := url.Parse(rawURI)
		if err != nil {
			continue
		}
		uriPath, err := url.PathUnescape(uri.Path)
		if err != nil || !strings.HasSuffix(uriPath, "/"+metadata.Filename) {
			continue
		}
		if !strings.EqualFold(fields[len(fields)-1], "SHA256:"+metadata.SHA256) {
			continue
		}
		return nil
	}
	return fmt.Errorf("APT did not resolve candidate %s to its authenticated PPA artifact", metadata.Version)
}

func isAllowedRepositoryURL(rawURL string) bool {
	uri, err := url.Parse(rawURL)
	if err != nil || uri.Scheme != "https" || uri.User != nil || uri.RawQuery != "" || uri.Fragment != "" {
		return false
	}
	basePath := "/mysteriumnetwork/node/ubuntu"
	return (uri.Host == "ppa.launchpadcontent.net" || uri.Host == "ppa.mysterium.network") &&
		(uri.Path == basePath || strings.HasPrefix(uri.Path, basePath+"/"))
}

func validPackageFilename(filename string) bool {
	return filename != "" &&
		!strings.HasPrefix(filename, "/") &&
		path.Clean(filename) == filename &&
		strings.HasPrefix(filename, "pool/") &&
		strings.HasSuffix(filename, ".deb")
}

func splitParagraphs(value string) []string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	return strings.Split(value, "\n\n")
}
