// Copyright 2022 AccelByte Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cors

import (
	"regexp"
	"strings"
)

// PatternType represents the type of CORS pattern.
type PatternType int

const (
	PatternTypeExact PatternType = iota
	PatternTypeWildcard
	PatternTypeRegex
)

// PatternMatcher matches origin values against configured CORS patterns.
// Supports exact matching, wildcard matching (*.domain.io), and regex matching (re:pattern).
type PatternMatcher struct {
	Pattern string
	Type    PatternType
	regex   *regexp.Regexp
}

// Compile creates a PatternMatcher from a pattern string.
// Patterns can be:
//   - "*" (matches any origin)
//   - "https://example.com" (exact match)
//   - "https://*.example.io" (wildcard subdomain match)
//   - "re:https://.*\.example\.com" (regex match)
//
// No structural validation is performed on wildcard patterns — validation
// of allowed values is the responsibility of the caller.
func Compile(pattern string) (*PatternMatcher, error) {
	// Check for regex prefix
	if strings.HasPrefix(pattern, "re:") {
		regexStr := strings.TrimPrefix(pattern, "re:")
		compiled, err := regexp.Compile(regexStr)
		if err != nil {
			return nil, newPatternCompilationError(pattern, err)
		}
		return &PatternMatcher{
			Pattern: pattern,
			Type:    PatternTypeRegex,
			regex:   compiled,
		}, nil
	}

	// Treat patterns containing "*" (but not bare "*") as wildcard patterns.
	// Bare "*" is kept as an exact-match sentinel that allows any origin.
	if strings.Contains(pattern, "*") && pattern != "*" {
		return &PatternMatcher{
			Pattern: pattern,
			Type:    PatternTypeWildcard,
		}, nil
	}

	// Exact match (including bare "*")
	return &PatternMatcher{
		Pattern: pattern,
		Type:    PatternTypeExact,
	}, nil
}

// MatchOrigin checks if the given origin matches this pattern.
func (pm *PatternMatcher) MatchOrigin(origin string) bool {
	if pm == nil {
		return false
	}
	switch pm.Type {
	case PatternTypeExact:
		return origin == pm.Pattern || pm.Pattern == "*"
	case PatternTypeWildcard:
		return pm.matchWildcard(origin)
	case PatternTypeRegex:
		if pm.regex == nil {
			return false
		}
		return pm.regex.MatchString(origin)
	default:
		return false
	}
}

// matchWildcard performs wildcard matching on the origin.
// The wildcard replaces exactly one subdomain label.
// Example: "https://*.example.io" matches "https://api.example.io"
// but not "https://a.b.example.io" or "https://example.io".
func (pm *PatternMatcher) matchWildcard(origin string) bool {
	// Find the position of the wildcard
	idx := strings.Index(pm.Pattern, "*")
	if idx == -1 {
		return false
	}

	// Pattern must be of the form <prefix>*.<suffix> to be useful.
	// If it doesn't have "*." we won't match anything meaningful, so return false.
	if idx+2 > len(pm.Pattern) || pm.Pattern[idx+1] != '.' {
		return false
	}

	prefix := pm.Pattern[:idx]        // e.g., "https://"
	suffix := pm.Pattern[idx+2:]      // e.g., "example.io" or "example.io:port"

	// Strip port from pattern suffix if present
	patternHostSuffix := suffix
	if colonIdx := strings.LastIndex(suffix, ":"); colonIdx != -1 {
		patternHostSuffix = suffix[:colonIdx]
	}

	// Extract protocol and host from origin
	originProtocol := ""
	originHostPort := origin
	if protocolIdx := strings.Index(origin, "://"); protocolIdx != -1 {
		originProtocol = origin[:protocolIdx+3]
		originHostPort = origin[protocolIdx+3:]
	}

	// Strip port from origin host
	originHost := originHostPort
	if colonIdx := strings.LastIndex(originHostPort, ":"); colonIdx != -1 {
		originHost = originHostPort[:colonIdx]
	}

	// Protocol must match exactly
	if originProtocol != prefix {
		return false
	}

	// Origin host must be exactly one subdomain level deeper than the pattern suffix.
	parts := strings.Split(originHost, ".")
	suffixParts := strings.Split(patternHostSuffix, ".")

	if len(parts) != len(suffixParts)+1 {
		return false
	}

	// Suffix labels must match (case-insensitive)
	for i := 0; i < len(suffixParts); i++ {
		if !strings.EqualFold(parts[i+1], suffixParts[i]) {
			return false
		}
	}

	// Subdomain label must be non-empty
	return parts[0] != ""
}
