// Copyright 2021 AccelByte Inc
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

package util

import "strings"

// GetDomain is used to get domain (scheme+host) from the specified URI
func GetDomain(uri string) string {
	afterDoubleSlashIndex := strings.Index(uri, "//")
	if afterDoubleSlashIndex == -1 {
		afterDoubleSlashIndex = 0
	} else {
		afterDoubleSlashIndex += 2
	}

	pathIndex := afterDoubleSlashIndex + strings.Index(uri[afterDoubleSlashIndex:], "/")
	if pathIndex > afterDoubleSlashIndex {
		return uri[:pathIndex]
	}
	return uri
}
