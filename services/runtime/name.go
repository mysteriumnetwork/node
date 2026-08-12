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

package runtime

import (
	"regexp"
	"strings"
)

var nameSanitizer = regexp.MustCompile(`[^a-z0-9_-]+`)

// NormalizeServiceType maps a runtime service name onto the canonical
// runtime-<name> service type. It is deliberately shared between the control
// plane and the service registry: both sides must agree on what "cdp",
// "CDP Browser" and "runtime-cdp" mean, or a registry entry could be
// unreachable by the name an operator uses for it. An empty string is returned
// when nothing usable remains.
func NormalizeServiceType(name string) string {
	normalized := strings.ToLower(strings.TrimSpace(name))
	normalized = nameSanitizer.ReplaceAllString(normalized, "-")
	normalized = strings.Trim(normalized, "-")
	if normalized == "" {
		return ""
	}
	if strings.HasPrefix(normalized, ServiceTypePrefix) {
		return normalized
	}
	return ServiceTypePrefix + normalized
}
