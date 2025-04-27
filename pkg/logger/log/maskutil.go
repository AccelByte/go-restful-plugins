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
	"github.com/sirupsen/logrus"
	"net/url"
	"regexp"
	"strings"
	"sync"
)

var FieldRegexCache = sync.Map{}

const (
	MaskedValue = "******"

	defaultMaskCharsCount = 2
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
	f.JsonPattern = regexp.MustCompile(fmt.Sprintf("\"%s\":(\\[.*?\\]|\".*?\")", fieldName))
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

// MaskPIIFields will mask the last n chars of field value on the content string based on the
// provided field name(s) in "fields" parameter separated by comma.
func MaskPIIFields(contentType, content, fields string) string {
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
			content = fieldRegex.JsonPattern.ReplaceAllStringFunc(content, func(m string) string {
				parts := fieldRegex.JsonPattern.FindStringSubmatch(m)
				if len(parts) < 2 {
					return m
				}
				matched := parts[1]
				if strings.HasPrefix(matched, "[") {
					matched = strings.Trim(matched, "[]")
					items := strings.Split(matched, ",")
					for i, item := range items {
						item = strings.TrimSpace(item)
						item = strings.Trim(item, "\"")
						items[i] = fmt.Sprintf("\"%s\"", maskLastNChars(item, defaultMaskCharsCount))
					}
					return fmt.Sprintf(`"%s":[%s]`, fieldName, strings.Join(items, ","))
				}
				return fmt.Sprintf(`"%s":"%s"`, fieldName, maskLastNChars(matched, defaultMaskCharsCount))
			})
		} else if strings.Contains(contentType, "application/x-www-form-urlencoded") {
			content = fieldRegex.QueryStringPattern.ReplaceAllStringFunc(content, func(m string) string {
				parts := fieldRegex.QueryStringPattern.FindStringSubmatch(m)
				if parts[1] != "" {
					return fieldName + "=" + maskLastNChars(parts[1], defaultMaskCharsCount)
				}
				return fieldName + "=" + maskLastNChars(parts[2], defaultMaskCharsCount)
			})
		} else {
			// try json pattern and form-data pattern
			if fieldRegex.JsonPattern.MatchString(content) {
				content = fieldRegex.JsonPattern.ReplaceAllStringFunc(content, func(m string) string {
					parts := fieldRegex.JsonPattern.FindStringSubmatch(m)
					if len(parts) < 2 {
						return m
					}
					matched := parts[1]
					if strings.HasPrefix(matched, "[") {
						matched = strings.Trim(matched, "[]")
						items := strings.Split(matched, ",")
						for i, item := range items {
							item = strings.TrimSpace(item)
							item = strings.Trim(item, "\"")
							items[i] = fmt.Sprintf("\"%s\"", maskLastNChars(item, defaultMaskCharsCount))
						}
						return fmt.Sprintf(`"%s":[%s]`, fieldName, strings.Join(items, ","))
					}
					return fmt.Sprintf(`"%s":"%s"`, fieldName, maskLastNChars(matched, defaultMaskCharsCount))
				})
			} else if fieldRegex.QueryStringPattern.MatchString(content) {
				content = fieldRegex.QueryStringPattern.ReplaceAllStringFunc(content, func(m string) string {
					parts := fieldRegex.QueryStringPattern.FindStringSubmatch(m)
					if parts[1] != "" {
						return fieldName + "=" + maskLastNChars(parts[1], defaultMaskCharsCount)
					}
					return fieldName + "=" + maskLastNChars(parts[2], defaultMaskCharsCount)
				})
			}
		}
	}

	return content
}

// MaskPIIQueryParams will mask the last n chars of field value on the uri based on the
// provided field name(s) in "fields" parameter separated by comma.
func MaskPIIQueryParams(uri string, fields string) string {
	if uri == "" || fields == "" {
		return uri
	}

	unescapedURI, err := url.QueryUnescape(uri)
	if err != nil {
		logrus.Warn("failed to unescape url parameter for masking sensitive data: ", err)
	} else {
		uri = unescapedURI
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

		uri = fieldRegex.QueryStringPattern.ReplaceAllStringFunc(uri, func(m string) string {
			parts := fieldRegex.QueryStringPattern.FindStringSubmatch(m)
			if parts[1] != "" {
				return fieldName + "=" + maskLastNChars(parts[1], defaultMaskCharsCount)
			}
			return fieldName + "=" + maskLastNChars(parts[2], defaultMaskCharsCount)
		})
	}
	return uri
}

func maskLastNChars(value string, n int) string {
	if value == "" {
		return ""
	}

	if strings.HasPrefix(value, `"`) {
		value = strings.Trim(value, `""`)
	}
	decoded, err := url.QueryUnescape(value)
	if err != nil {
		logrus.Warn("failed to unescape url parameter for masking sensitive data: ", err)
	} else {
		value = decoded
	}
	if strings.Contains(value, ",") {
		maskedValues := make([]string, 0)
		values := strings.Split(value, ",")
		for _, v := range values {
			maskedValues = append(maskedValues, mask(v, n))
		}
		return strings.Join(maskedValues, ",")
	} else {
		return mask(value, n)
	}
}

func mask(value string, n int) string {
	if value == "" {
		return ""
	}
	if strings.Contains(value, "@") {
		parts := strings.Split(value, "@")
		if len(parts[0]) <= n {
			return fmt.Sprintf("%s@%s", parts[0][:1]+"****", parts[1])
		}
		return fmt.Sprintf("%s@%s", parts[0][:len(parts[0])-n]+"****", parts[1])
	}

	if len(value) <= n {
		return value[:1] + "****"
	}
	return value[:len(value)-n] + "****"
}
