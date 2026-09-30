/*
 * Copyright (C) 2026 The "MysteriumNetwork/node" Authors.
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
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	testProviderID = "0x1111111111111111111111111111111111111111"
	testConsumerID = "0x2222222222222222222222222222222222222222"
)

func traceContext(stage string) sessionTraceContext {
	return sessionTraceContext{
		Stage: stage,
		sessionContext: sessionContext{
			ID:       "session-id",
			Consumer: testConsumerID,
			Provider: testProviderID,
		},
	}
}

// Provider-side trace stages must be owned (and therefore signed) by the provider.
// A provider node does not hold the consumer's key, so attributing a provider stage
// to the consumer makes signing fail with keystore.ErrLocked.
func TestTraceEventToMetricsEvent_ProviderStageOwnedByProvider(t *testing.T) {
	for _, stage := range []string{
		"Provider session create",
		"Provider session validation",
		"Provider P2P exchange",
	} {
		owner, event := traceEventToMetricsEvent(traceContext(stage))

		assert.Equal(t, testProviderID, owner, stage)
		assert.True(t, event.IsProvider, stage)
		assert.Equal(t, testConsumerID, event.TargetId, stage)
	}
}

func TestTraceEventToMetricsEvent_ConsumerStageOwnedByConsumer(t *testing.T) {
	owner, event := traceEventToMetricsEvent(traceContext("Consumer session creation"))

	assert.Equal(t, testConsumerID, owner)
	assert.False(t, event.IsProvider)
	assert.Equal(t, testProviderID, event.TargetId)
}

// traceEventToMetricsEvent derives the event owner from the stage name prefix.
// Every trace stage must therefore start with "Provider" or "Consumer", otherwise
// a provider-side stage is silently attributed to the consumer.
func TestTraceStageNames_HaveOwnerPrefix(t *testing.T) {
	stageLiteral := regexp.MustCompile(`StartStage\("([^"]*)"\)`)
	root := filepath.Join("..", "..")

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", "vendor", "node_modules", "build":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, m := range stageLiteral.FindAllStringSubmatch(string(src), -1) {
			stage := m[1]
			assert.True(t,
				strings.HasPrefix(stage, "Provider") || strings.HasPrefix(stage, "Consumer"),
				"trace stage %q in %s must start with \"Provider\" or \"Consumer\"", stage, path,
			)
		}
		return nil
	})
	assert.NoError(t, err)
}
