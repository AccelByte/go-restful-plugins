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

package legal

import (
	"net/http"

	"github.com/AccelByte/iam-go-sdk"
	"github.com/AccelByte/legal-go-sdk"
	"github.com/emicklei/go-restful/v3"
	"github.com/sirupsen/logrus"
)

const claimsAttribute = "JWTClaims"

// Filter handles eligibility using filter
type Filter struct {
	legalClient legal.LegalClient
}

// ErrorResponse is the generic structure for communicating errors from a REST endpoint.
type ErrorResponse struct {
	ErrorCode    int    `json:"errorCode"`
	ErrorMessage string `json:"errorMessage"`
}

// NewFilter creates new Filter instance
func NewFilter(client legal.LegalClient) *Filter {
	return &Filter{legalClient: client}
}

func logIfErr(err error) {
	if err != nil {
		logrus.Error(err)
	}
}

func (filter *Filter) Eligibility() restful.FilterFunction {
	return func(request *restful.Request, response *restful.Response, chain *restful.FilterChain) {
		claims, _ := request.Attribute(claimsAttribute).(*iam.JWTClaims)
		if claims == nil {
			logrus.Warn("unauthorized access: no JWT claims found")
			logIfErr(response.WriteHeaderAndJson(http.StatusUnauthorized, ErrorResponse{
				ErrorCode:    UnauthorizedAccess,
				ErrorMessage: ErrorCodeMapping[UnauthorizedAccess],
			}, restful.MIME_JSON))

			return
		}

		valid, _ := filter.legalClient.ValidatePolicyVersions(claims)

		if !valid {
			logrus.Warn("forbidden access: user not sign all crucial mandatory policy version")
			logIfErr(response.WriteHeaderAndJson(http.StatusForbidden, ErrorResponse{
				ErrorCode:    UserNotEligible,
				ErrorMessage: ErrorCodeMapping[UserNotEligible],
			}, restful.MIME_JSON))

			return
		}

		chain.ProcessFilter(request, response)
	}
}
