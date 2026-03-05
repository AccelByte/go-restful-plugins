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

import "fmt"

// configFetchError is returned when fetching CORS config from the config service fails.
type configFetchError struct {
	namespace string
	err       error
}

func (e *configFetchError) Error() string {
	return fmt.Sprintf("failed to fetch CORS config for namespace %q: %v", e.namespace, e.err)
}

func (e *configFetchError) Unwrap() error {
	return e.err
}

// patternCompilationError is returned when a CORS pattern (regex or wildcard) fails to compile.
type patternCompilationError struct {
	pattern string
	err     error
}

func (e *patternCompilationError) Error() string {
	return fmt.Sprintf("failed to compile CORS pattern %q: %v", e.pattern, e.err)
}

func (e *patternCompilationError) Unwrap() error {
	return e.err
}

// newConfigFetchError creates a configFetchError.
func newConfigFetchError(namespace string, err error) error {
	return &configFetchError{namespace: namespace, err: err}
}

// newPatternCompilationError creates a patternCompilationError.
func newPatternCompilationError(pattern string, err error) error {
	return &patternCompilationError{pattern: pattern, err: err}
}

