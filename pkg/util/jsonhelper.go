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

import (
	"bytes"
	"encoding/json"
	"strings"

	"github.com/sirupsen/logrus"
)

// MinifyJSON is used to compact the json bytes (e.g. removing whitespaces and new lines character)
func MinifyJSON(jsonBytes []byte) string {
	buffer := new(bytes.Buffer)
	if err := json.Compact(buffer, jsonBytes); err != nil {
		logrus.Warnf("Fail to compact json: %v", err)

		// the uncompleted json will throw an error, i.e. {"foo":"bar"
		// so instead, switch with the simple parse (e.g. removing new lines)
		jsonString := string(jsonBytes)
		jsonString = strings.ReplaceAll(jsonString, "\n", "")
		jsonString = strings.ReplaceAll(jsonString, "\r", "")
		return jsonString
	}
	return buffer.String()
}
