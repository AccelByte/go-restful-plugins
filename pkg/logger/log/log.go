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
	"github.com/emicklei/go-restful/v3"
)

const (
	MaskedQueryParamsAttribute    = "MaskedQueryParams"
	MaskedRequestFieldsAttribute  = "MaskedRequestFields"
	MaskedResponseFieldsAttribute = "MaskedResponseFields"
	UserIDAttribute               = "LogUserId"
	ClientIDAttribute             = "LogClientId"
	NamespaceAttribute            = "LogNamespace"
)

// Option contains attribute options for log functionality
type Option struct {
	// Query param that need to masked in url, separated with comma
	MaskedQueryParams string
	// Field that need to masked in request body, separated with comma
	MaskedRequestFields string
	// Field that need to masked in response body, separated with comma
	MaskedResponseFields string
}

// Attribute filter is used to define the log attribute for the endpoint.
func Attribute(option Option) restful.FilterFunction {
	return func(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
		if option.MaskedQueryParams != "" {
			req.SetAttribute(MaskedQueryParamsAttribute, option.MaskedQueryParams)
		}
		if option.MaskedRequestFields != "" {
			req.SetAttribute(MaskedRequestFieldsAttribute, option.MaskedRequestFields)
		}
		if option.MaskedResponseFields != "" {
			req.SetAttribute(MaskedResponseFieldsAttribute, option.MaskedResponseFields)
		}
		chain.ProcessFilter(req, resp)
	}
}
