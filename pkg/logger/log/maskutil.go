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

package log

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
)

var FieldRegexCache = sync.Map{}

const (
	MaskedValue = "******"
)

// FieldRegex contains regex patterns for field name in varied content-types.
type FieldRegex struct {
	FieldName          string
	JsonPattern        *regexp.Regexp
	QueryStringPattern *regexp.Regexp
}

// InitFieldRegex initialize the FieldRegex along with its regex patterns.
func (f *FieldRegex) InitFieldRegex(fieldName string) {
	f.FieldName = fieldName
	// "fieldName":"(.*?)"
	f.JsonPattern = regexp.MustCompile(fmt.Sprintf("\"%s\":\"(.*?)\"", fieldName))
	// fieldName=(.*?[^&]*)|fieldName=(.*?)$
	f.QueryStringPattern = regexp.MustCompile(fmt.Sprintf("%s=(.*?[^&]*)|%s=(.*?)$", fieldName, fieldName))
}

// MaskFields will mask the field value on the content string based on the
// provided field name(s) in "fields" parameter separated by comma.
func MaskFields(contentType, content, fields string) string {
	if content == "" || fields == "" {
		return content
	}

	fieldNames := strings.Split(fields, ",")
	for _, fieldName := range fieldNames {
		var fieldRegex FieldRegex
		if val, ok := FieldRegexCache.Load(fieldName); ok {
			fieldRegex = val.(FieldRegex)
		} else {
			fieldRegex = FieldRegex{}
			fieldRegex.InitFieldRegex(fieldName)
			FieldRegexCache.Store(fieldName, fieldRegex)
		}

		if strings.Contains(contentType, "application/json") {
			content = fieldRegex.JsonPattern.ReplaceAllString(content, fmt.Sprintf("\"%s\":\"%s\"", fieldName, MaskedValue))
		} else if strings.Contains(contentType, "application/x-www-form-urlencoded") {
			content = fieldRegex.QueryStringPattern.ReplaceAllString(content, fmt.Sprintf("%s=%s", fieldName, MaskedValue))
		} else {
			// try json pattern and form-data pattern
			if fieldRegex.JsonPattern.MatchString(content) {
				content = fieldRegex.JsonPattern.ReplaceAllString(content, fmt.Sprintf("\"%s\":\"%s\"", fieldName, MaskedValue))
			} else if fieldRegex.QueryStringPattern.MatchString(content) {
				content = fieldRegex.QueryStringPattern.ReplaceAllString(content, fmt.Sprintf("%s=%s", fieldName, MaskedValue))
			}
		}
	}

	return content
}

// MaskQueryParams will mask the field value on the uri based on the
// provided field name(s) in "fields" parameter separated by comma.
func MaskQueryParams(uri string, fields string) string {
	if uri == "" || fields == "" {
		return uri
	}

	fieldNames := strings.Split(fields, ",")
	for _, fieldName := range fieldNames {
		var fieldRegex FieldRegex
		if val, ok := FieldRegexCache.Load(fieldName); ok {
			fieldRegex = val.(FieldRegex)
		} else {
			fieldRegex = FieldRegex{}
			fieldRegex.InitFieldRegex(fieldName)
			FieldRegexCache.Store(fieldName, fieldRegex)
		}

		uri = fieldRegex.QueryStringPattern.ReplaceAllString(uri, fmt.Sprintf("%s=%s", fieldName, MaskedValue))
	}
	return uri
}
