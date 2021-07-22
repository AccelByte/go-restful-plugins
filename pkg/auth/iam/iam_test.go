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

package iam

import (
	"net/http"
	"testing"

	"github.com/AccelByte/go-restful-plugins/v4/pkg/constant"
	"github.com/AccelByte/iam-go-sdk"
	"github.com/emicklei/go-restful/v3"
	"github.com/stretchr/testify/assert"
)

// nolint:paralleltest
func TestValidateRefererHeader(t *testing.T) {
	iamClient := iam.NewMockClient()
	filter := NewFilter(iamClient)
	userTokenClaims, _ := filter.iamClient.ValidateAndParseClaims("dummyToken")

	correctRequest1 := &restful.Request{
		Request: &http.Request{
			Header: map[string][]string{
				constant.Referer: {"http://127.0.0.1"},
			},
		},
	}
	assert.Equal(t, true, filter.validateRefererHeader(correctRequest1, userTokenClaims))

	correctRequest2 := &restful.Request{
		Request: &http.Request{
			Header: map[string][]string{
				constant.Referer: {"http://127.0.0.1/path/path"},
			},
		},
	}
	assert.Equal(t, true, filter.validateRefererHeader(correctRequest2, userTokenClaims))

	incorrectRequest1 := &restful.Request{
		Request: &http.Request{
			Header: map[string][]string{
				constant.Referer: {"http://127.0.0.2"},
			},
		},
	}
	assert.Equal(t, false, filter.validateRefererHeader(incorrectRequest1, userTokenClaims))

	incorrectRequest2 := &restful.Request{
		Request: &http.Request{
			Header: map[string][]string{
				constant.Referer: {""},
			},
		},
	}
	assert.Equal(t, false, filter.validateRefererHeader(incorrectRequest2, userTokenClaims))
}

// nolint:paralleltest
func TestValidateRefererHeaderWithCustomFilterOptions(t *testing.T) {
	iamClient := iam.NewMockClient()

	// Test Filter With Nil Options
	filterWithNilOptions := NewFilterWithOptions(iamClient, nil)
	userTokenClaims, _ := filterWithNilOptions.iamClient.ValidateAndParseClaims("dummyToken")

	correctRequest := &restful.Request{
		Request: &http.Request{
			Header: map[string][]string{
				constant.Referer: {"http://127.0.0.1"},
			},
		},
	}
	assert.Equal(t, true, filterWithNilOptions.validateRefererHeader(correctRequest, userTokenClaims))

	// Test Filter With Strict Referer Header Validation
	filterWithStrictRefererHeaderValidation := NewFilterWithOptions(iamClient, &FilterInitializationOptions{StrictRefererHeaderValidation: true})
	userTokenClaims2, _ := filterWithStrictRefererHeaderValidation.iamClient.ValidateAndParseClaims("dummyToken")

	correctRequest2 := &restful.Request{
		Request: &http.Request{
			Header: map[string][]string{
				constant.Referer: {"http://127.0.0.1"},
			},
		},
	}
	assert.Equal(t, true, filterWithStrictRefererHeaderValidation.validateRefererHeader(correctRequest2, userTokenClaims2))
}
