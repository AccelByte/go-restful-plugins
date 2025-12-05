// Copyright 2021-2025 AccelByte Inc
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
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/AccelByte/go-restful-plugins/v4/pkg/constant"
	"github.com/AccelByte/iam-go-sdk/v2"
	"github.com/emicklei/go-restful/v3"
	"github.com/stretchr/testify/assert"
)

// nolint:paralleltest
func TestValidateRefererHeader_RedirectUriIsDomain(t *testing.T) {
	iamClient := &iam.MockClient{
		Healthy:     true,
		RedirectURI: "https://www.example.com",
	}
	filter := NewFilter(iamClient)

	testcases := []struct {
		name          string
		refererHeader string
		allowed       bool
	}{
		{
			name:          "normal referer",
			refererHeader: "https://www.example.com",
			allowed:       true,
		},
		{
			name:          "normal referer with path",
			refererHeader: "https://www.example.com/path/path",
			allowed:       true,
		},
		{
			name:          "wrong referer",
			refererHeader: "https://www.example.net",
			allowed:       false,
		},
		{
			name:          "wrong referer 2nd",
			refererHeader: "https://wrong.example.com",
			allowed:       false,
		},
		{
			name:          "wrong referer 3rd",
			refererHeader: "https://www.wrong.com",
			allowed:       false,
		},
		{
			name:          "wrong referer 4th",
			refererHeader: "https://example.com",
			allowed:       false,
		},
		{
			name:          "wrong referer 5th",
			refererHeader: "https://www.example.com:8080",
			allowed:       false,
		},
		{
			name:          "wrong referer 6th",
			refererHeader: "https://subdomain.example.com",
			allowed:       false,
		},
		{
			name:          "referer with extra wrong domain",
			refererHeader: "https://www.example.com.something.net",
			allowed:       false,
		},
		{
			name:          "empty referer",
			refererHeader: "",
			allowed:       false,
		},
	}

	for _, testcase := range testcases {
		t.Run(testcase.name, func(t *testing.T) {
			userTokenClaims, _ := filter.iamClient.ValidateAndParseClaims("dummyToken")

			correctRequest := &restful.Request{
				Request: &http.Request{
					Header: map[string][]string{
						constant.Referer: {testcase.refererHeader},
					},
				},
			}

			actual := filter.validateRefererHeader(correctRequest, userTokenClaims, false)
			assert.Equal(t, testcase.allowed, actual)
		})
	}
}

// nolint:paralleltest
func TestValidateRefererHeader_RedirectUriIsIP(t *testing.T) {
	iamClient := &iam.MockClient{
		Healthy:     true,
		RedirectURI: "http://127.0.0.1",
	}
	filter := NewFilter(iamClient)

	testcases := []struct {
		name          string
		refererHeader string
		allowed       bool
	}{
		{
			name:          "normal referer",
			refererHeader: "http://127.0.0.1",
			allowed:       true,
		},
		{
			name:          "normal referer with path",
			refererHeader: "http://127.0.0.1/path/path",
			allowed:       true,
		},
		{
			name:          "wrong referer",
			refererHeader: "http://127.0.0.2",
			allowed:       false,
		},
		{
			name:          "wrong referer 2nd",
			refererHeader: "127.0.0.1",
			allowed:       false,
		},
		{
			name:          "wrong referer 3rd",
			refererHeader: "https://subdomain.127.0.0.1",
			allowed:       false,
		},
		{
			name:          "referer with extra wrong IP",
			refererHeader: "http://127.0.0.1.2",
			allowed:       false,
		},
		{
			name:          "empty referer",
			refererHeader: "",
			allowed:       false,
		},
	}

	for _, testcase := range testcases {
		t.Run(testcase.name, func(t *testing.T) {
			userTokenClaims, _ := filter.iamClient.ValidateAndParseClaims("dummyToken")

			correctRequest := &restful.Request{
				Request: &http.Request{
					Header: map[string][]string{
						constant.Referer: {testcase.refererHeader},
					},
				},
			}

			actual := filter.validateRefererHeader(correctRequest, userTokenClaims, false)

			assert.Equal(t, testcase.allowed, actual)
		})
	}
}

// nolint:paralleltest
func TestValidateRefererHeader_RedirectUriContainsPort(t *testing.T) {
	iamClient := &iam.MockClient{
		Healthy:     true,
		RedirectURI: "https://www.example.com:8080",
	}
	filter := NewFilter(iamClient)

	testcases := []struct {
		name          string
		refererHeader string
		allowed       bool
	}{
		{
			name:          "normal referer",
			refererHeader: "https://www.example.com:8080",
			allowed:       true,
		},
		{
			name:          "normal referer with path",
			refererHeader: "https://www.example.com:8080/path/path",
			allowed:       true,
		},
		{
			name:          "wrong referer",
			refererHeader: "https://www.example.com",
			allowed:       false,
		},
		{
			name:          "wrong referer 2nd",
			refererHeader: "https://wrong.example.com",
			allowed:       false,
		},
		{
			name:          "wrong referer 3rd",
			refererHeader: "https://www.wrong.com",
			allowed:       false,
		},
		{
			name:          "wrong referer 4th",
			refererHeader: "www.example.com:8080",
			allowed:       false,
		},
		{
			name:          "wrong referer 5th",
			refererHeader: "https://subdomain.example.com:8080",
			allowed:       false,
		},
		{
			name:          "referer with extra wrong domain",
			refererHeader: "https://www.example.com.something.net:8080",
			allowed:       false,
		},
	}

	for _, testcase := range testcases {
		t.Run(testcase.name, func(t *testing.T) {
			userTokenClaims, _ := filter.iamClient.ValidateAndParseClaims("dummyToken")

			correctRequest := &restful.Request{
				Request: &http.Request{
					Header: map[string][]string{
						constant.Referer: {testcase.refererHeader},
					},
				},
			}

			actual := filter.validateRefererHeader(correctRequest, userTokenClaims, false)
			assert.Equal(t, testcase.allowed, actual)
		})
	}
}

// nolint:paralleltest
func TestValidateRefererHeader_ClientHaveMultipleRedirectURIs(t *testing.T) {
	iamClientWithMultipleRedirectURI := &iam.MockClient{
		Healthy:     true,
		RedirectURI: "https://www.example.com,https://www.example.io",
	}
	filter := NewFilter(iamClientWithMultipleRedirectURI)

	testcases := []struct {
		name          string
		refererHeader string
		allowed       bool
	}{
		{
			name:          "normal referer",
			refererHeader: "https://www.example.com",
			allowed:       true,
		},
		{
			name:          "normal referer 2nd",
			refererHeader: "https://www.example.io",
			allowed:       true,
		},
		{
			name:          "normal referer with path",
			refererHeader: "https://www.example.com/path/path",
			allowed:       true,
		},
		{
			name:          "normal referer with path 2nd",
			refererHeader: "https://www.example.io/path/path",
			allowed:       true,
		},
		{
			name:          "wrong referer",
			refererHeader: "https://www.example.net",
			allowed:       false,
		},
		{
			name:          "wrong referer 2nd",
			refererHeader: "https://wrong.example.com",
			allowed:       false,
		},
		{
			name:          "wrong referer 3rd",
			refererHeader: "https://www.wrong.com",
			allowed:       false,
		},
		{
			name:          "wrong referer 4th",
			refererHeader: "www.example.com",
			allowed:       false,
		},
		{
			name:          "wrong referer 5th",
			refererHeader: "https://www.example.com:8080",
			allowed:       false,
		},
		{
			name:          "wrong referer 6th",
			refererHeader: "https://subdomaim.example.com",
			allowed:       false,
		},
		{
			name:          "referer with extra wrong domain",
			refererHeader: "https://www.example.com.something.net",
			allowed:       false,
		},
		{
			name:          "empty referer",
			refererHeader: "",
			allowed:       false,
		},
	}

	for _, testcase := range testcases {
		t.Run(testcase.name, func(t *testing.T) {
			userTokenClaims, _ := filter.iamClient.ValidateAndParseClaims("dummyToken")

			correctRequest := &restful.Request{
				Request: &http.Request{
					Header: map[string][]string{
						constant.Referer: {testcase.refererHeader},
					},
				},
			}

			actual := filter.validateRefererHeader(correctRequest, userTokenClaims, false)
			assert.Equal(t, testcase.allowed, actual)
		})
	}
}

// nolint:paralleltest
func TestValidateRefererHeader_StrictRefererValidation(t *testing.T) {
	iamClient := &iam.MockClient{
		Healthy:     true,
		RedirectURI: "https://www.example.com/admin",
	}
	filter := NewFilterWithOptions(iamClient, &FilterInitializationOptions{StrictRefererHeaderValidation: true})

	testcases := []struct {
		name          string
		refererHeader string
		allowed       bool
	}{
		{
			name:          "normal referer",
			refererHeader: "https://www.example.com/admin",
			allowed:       true,
		},
		{
			name:          "normal referer with path",
			refererHeader: "https://www.example.com/admin/path/path",
			allowed:       true,
		},
		{
			name:          "wrong referer with path",
			refererHeader: "https://www.example.com/path/path",
			allowed:       false,
		},
		{
			name:          "wrong referer",
			refererHeader: "https://www.example.net",
			allowed:       false,
		},
		{
			name:          "wrong referer 2nd",
			refererHeader: "https://wrong.example.com",
			allowed:       false,
		},
		{
			name:          "wrong referer 3rd",
			refererHeader: "https://www.wrong.com",
			allowed:       false,
		},
		{
			name:          "wrong referer 4th",
			refererHeader: "https://www.example.com:8080/admin",
			allowed:       false,
		},
		{
			name:          "wrong referer 5th",
			refererHeader: "https://subdomain.example.com/admin",
			allowed:       false,
		},
		{
			name:          "referer with extra wrong domain",
			refererHeader: "https://www.example.com.something.net/admin",
			allowed:       false,
		},
		{
			name:          "empty referer",
			refererHeader: "",
			allowed:       false,
		},
	}

	for _, testcase := range testcases {
		t.Run(testcase.name, func(t *testing.T) {
			userTokenClaims, _ := filter.iamClient.ValidateAndParseClaims("dummyToken")

			correctRequest := &restful.Request{
				Request: &http.Request{
					Header: map[string][]string{
						constant.Referer: {testcase.refererHeader},
					},
				},
			}

			actual := filter.validateRefererHeader(correctRequest, userTokenClaims, false)
			assert.Equal(t, testcase.allowed, actual)
		})
	}
}

// nolint:paralleltest
func TestValidateRefererHeaderWithoutSubdomain_SimpleRedirectURI(t *testing.T) {
	iamClient := &iam.MockClient{
		Healthy:     true,
		RedirectURI: "https://example.com",
	}
	filter := NewFilterWithOptions(iamClient, &FilterInitializationOptions{AllowSubdomainMatchRefererHeaderValidation: true})

	testcases := []struct {
		name          string
		refererHeader string
		allowed       bool
	}{
		{
			name:          "normal referer",
			refererHeader: "https://example.com",
			allowed:       true,
		},
		{
			name:          "normal referer",
			refererHeader: "https://www.example.com",
			allowed:       true,
		},
		{
			name:          "normal referer",
			refererHeader: "https://www.example.com/admin/path",
			allowed:       true,
		},
		{
			name:          "normal referer",
			refererHeader: "https://subdomain.example.com",
			allowed:       true,
		},
		{
			name:          "normal referer",
			refererHeader: "https://subdomain.example.com/admin/path",
			allowed:       true,
		},
		{
			name:          "wrong referer",
			refererHeader: "https://example.net",
			allowed:       false,
		},
		{
			name:          "wrong referer",
			refererHeader: "http://example.com",
			allowed:       false,
		},
		{
			name:          "wrong referer",
			refererHeader: "https://www.wrong.com",
			allowed:       false,
		},
		{
			name:          "wrong referer",
			refererHeader: "https://subdomain.wrong.com",
			allowed:       false,
		},
		{
			name:          "wrong referer",
			refererHeader: "https://subdomain.example.com.something.com",
			allowed:       false,
		},
		{
			name:          "wrong referer",
			refererHeader: "https://example.com.something.com",
			allowed:       false,
		},
		{
			name:          "wrong referer",
			refererHeader: "https://examplewww.com",
			allowed:       false,
		},
		{
			name:          "wrong referer",
			refererHeader: "https://www.examplewww.com",
			allowed:       false,
		},
	}

	for _, testcase := range testcases {
		t.Run(testcase.name, func(t *testing.T) {
			userTokenClaims, _ := filter.iamClient.ValidateAndParseClaims("dummyToken")

			correctRequest := &restful.Request{
				Request: &http.Request{
					Header: map[string][]string{
						constant.Referer: {testcase.refererHeader},
					},
				},
			}

			actual := filter.validateRefererHeader(correctRequest, userTokenClaims, false)

			assert.Equal(t, testcase.allowed, actual)
		})
	}
}

// nolint:paralleltest
func TestValidateRefererHeaderWithSubdomain_SimpleRedirectURI(t *testing.T) {
	iamClient := &iam.MockClient{
		Healthy:     true,
		RedirectURI: "https://example.com",
	}
	filter := NewFilterWithOptions(iamClient, &FilterInitializationOptions{AllowSubdomainMatchRefererHeaderValidation: true, SubdomainValidationEnabled: true})

	testcases := []struct {
		name          string
		refererHeader string
		allowed       bool
	}{
		{
			name:          "normal referer with subdomain",
			refererHeader: "https://mock.example.com",
			allowed:       true,
		},
		{
			name:          "normal referer",
			refererHeader: "https://mock.example.com/admin/path",
			allowed:       true,
		},
		{
			name:          "wrong subdomain",
			refererHeader: "https://subdomain.example.com",
			allowed:       false,
		},
		{
			name:          "wrong subdomain",
			refererHeader: "https://subdomain.example.com/admin/path",
			allowed:       false,
		},
		{
			name:          "wrong referer",
			refererHeader: "https://example.net",
			allowed:       false,
		},
		{
			name:          "wrong referer",
			refererHeader: "http://example.com",
			allowed:       false,
		},
		{
			name:          "wrong referer",
			refererHeader: "https://www.wrong.com",
			allowed:       false,
		},
		{
			name:          "wrong referer",
			refererHeader: "https://subdomain.wrong.com",
			allowed:       false,
		},
		{
			name:          "wrong referer",
			refererHeader: "https://subdomain.example.com.something.com",
			allowed:       false,
		},
		{
			name:          "wrong referer",
			refererHeader: "https://example.com.something.com",
			allowed:       false,
		},
		{
			name:          "wrong referer",
			refererHeader: "https://examplewww.com",
			allowed:       false,
		},
		{
			name:          "wrong referer",
			refererHeader: "https://www.examplewww.com",
			allowed:       false,
		},
	}

	for _, testcase := range testcases {
		t.Run(testcase.name, func(t *testing.T) {
			userTokenClaims, _ := filter.iamClient.ValidateAndParseClaims("dummyToken")

			correctRequest := &restful.Request{
				Request: &http.Request{
					Header: map[string][]string{
						constant.Referer: {testcase.refererHeader},
					},
				},
			}

			actual := filter.validateRefererHeader(correctRequest, userTokenClaims, false)

			assert.Equal(t, testcase.allowed, actual)
		})
	}
}

func TestValidateRefererHeaderWithSubdomain_SimpleRedirectURISkipSubdomainValidation(t *testing.T) {
	iamClient := &iam.MockClient{
		Healthy:     true,
		RedirectURI: "https://dev.example.com,https://example.com",
	}
	filter := NewFilterWithOptions(iamClient, &FilterInitializationOptions{AllowSubdomainMatchRefererHeaderValidation: true, SubdomainValidationEnabled: true})

	testcases := []struct {
		name          string
		refererHeader string
		allowed       bool
	}{
		{
			name:          "normal referer with subdomain",
			refererHeader: "https://dev.example.com",
			allowed:       true,
		},
		{
			name:          "normal referer with subdomain",
			refererHeader: "https://example.com",
			allowed:       true,
		},
		{
			name:          "normal referer",
			refererHeader: "https://example.com/admin/path",
			allowed:       true,
		},
		{
			name:          "normal referer with subdomain",
			refererHeader: "https://mock.example.com",
			allowed:       true,
		},
		{
			name:          "incorrect domain",
			refererHeader: "https://mock.com",
			allowed:       false,
		},
		{
			name:          "incorrect domain",
			refererHeader: "https://mock.example.net",
			allowed:       false,
		},
		{
			name:          "normal referer with subdomain",
			refererHeader: "https://mock.example.com/admin/path",
			allowed:       true,
		},
		{
			name:          "different subdomain",
			refererHeader: "https://subdomain.example.com",
			allowed:       true,
		},
		{
			name:          "different subdomain",
			refererHeader: "https://subdomain.example.com/admin/path",
			allowed:       true,
		},
		{
			name:          "wrong referer",
			refererHeader: "https://example.net",
			allowed:       false,
		},
		{
			name:          "wrong referer",
			refererHeader: "http://example.com",
			allowed:       false,
		},
		{
			name:          "wrong referer",
			refererHeader: "https://www.wrong.com",
			allowed:       false,
		},
		{
			name:          "wrong referer",
			refererHeader: "https://subdomain.wrong.com",
			allowed:       false,
		},
		{
			name:          "wrong referer",
			refererHeader: "https://subdomain.example.com.something.com",
			allowed:       false,
		},
		{
			name:          "wrong referer",
			refererHeader: "https://example.com.something.com",
			allowed:       false,
		},
		{
			name:          "wrong referer",
			refererHeader: "https://examplewww.com",
			allowed:       false,
		},
		{
			name:          "wrong referer",
			refererHeader: "https://www.examplewww.com",
			allowed:       false,
		},
	}

	for _, testcase := range testcases {
		t.Run(testcase.name, func(t *testing.T) {
			userTokenClaims, _ := filter.iamClient.ValidateAndParseClaims("dummyToken")

			correctRequest := &restful.Request{
				Request: &http.Request{
					Header: map[string][]string{
						constant.Referer: {testcase.refererHeader},
					},
				},
			}

			actual := filter.validateRefererHeader(correctRequest, userTokenClaims, true)

			assert.Equal(t, testcase.allowed, actual)
		})
	}
}

// nolint:paralleltest
func TestValidateRefererHeaderWithoutSubdomain_SimpleRedirectURIContainsWWW(t *testing.T) {
	iamClient := &iam.MockClient{
		Healthy:     true,
		RedirectURI: "https://www.example.com",
	}
	filter := NewFilterWithOptions(iamClient, &FilterInitializationOptions{AllowSubdomainMatchRefererHeaderValidation: true})

	testcases := []struct {
		name          string
		refererHeader string
		allowed       bool
	}{
		{
			name:          "normal referer",
			refererHeader: "https://example.com",
			allowed:       true,
		},
		{
			name:          "normal referer",
			refererHeader: "https://www.example.com",
			allowed:       true,
		},
		{
			name:          "normal referer",
			refererHeader: "https://www.example.com/admin/path",
			allowed:       true,
		},
		{
			name:          "normal referer",
			refererHeader: "https://subdomain.example.com",
			allowed:       true,
		},
		{
			name:          "normal referer",
			refererHeader: "https://subdomain.example.com/admin/path",
			allowed:       true,
		},
		{
			name:          "wrong referer",
			refererHeader: "https://example.net",
			allowed:       false,
		},
		{
			name:          "wrong referer",
			refererHeader: "http://example.com",
			allowed:       false,
		},
		{
			name:          "wrong referer",
			refererHeader: "https://www.wrong.com",
			allowed:       false,
		},
		{
			name:          "wrong referer",
			refererHeader: "https://subdomain.wrong.com",
			allowed:       false,
		},
		{
			name:          "wrong referer",
			refererHeader: "https://subdomain.example.com.something.com",
			allowed:       false,
		},
		{
			name:          "wrong referer",
			refererHeader: "https://example.com.something.com",
			allowed:       false,
		},
		{
			name:          "wrong referer",
			refererHeader: "https://examplewww.com",
			allowed:       false,
		},
		{
			name:          "wrong referer",
			refererHeader: "https://www.examplewww.com",
			allowed:       false,
		},
	}

	for _, testcase := range testcases {
		t.Run(testcase.name, func(t *testing.T) {
			userTokenClaims, _ := filter.iamClient.ValidateAndParseClaims("dummyToken")

			correctRequest := &restful.Request{
				Request: &http.Request{
					Header: map[string][]string{
						constant.Referer: {testcase.refererHeader},
					},
				},
			}

			actual := filter.validateRefererHeader(correctRequest, userTokenClaims, false)

			assert.Equal(t, testcase.allowed, actual)
		})
	}
}

// nolint:paralleltest
func TestValidateRefererHeaderWithoutSubdomain_RedirectURIContainsPort(t *testing.T) {
	iamClient := &iam.MockClient{
		Healthy:     true,
		RedirectURI: "https://www.example.com:8080",
	}
	filter := NewFilterWithOptions(iamClient, &FilterInitializationOptions{AllowSubdomainMatchRefererHeaderValidation: true})

	testcases := []struct {
		name          string
		refererHeader string
		allowed       bool
	}{
		{
			name:          "normal referer",
			refererHeader: "https://example.com:8080",
			allowed:       true,
		},
		{
			name:          "normal referer",
			refererHeader: "https://www.example.com:8080",
			allowed:       true,
		},
		{
			name:          "normal referer",
			refererHeader: "https://www.example.com:8080/admin/path",
			allowed:       true,
		},
		{
			name:          "normal referer",
			refererHeader: "https://subdomain.example.com:8080",
			allowed:       true,
		},
		{
			name:          "normal referer",
			refererHeader: "https://subdomain.example.com:8080/admin/path",
			allowed:       true,
		},
		{
			name:          "wrong referer",
			refererHeader: "https://example.com",
			allowed:       false,
		},
		{
			name:          "wrong referer",
			refererHeader: "https://www.example.com:8081",
			allowed:       false,
		},
		{
			name:          "wrong referer",
			refererHeader: "https://subdomain.example.com:8081",
			allowed:       false,
		},
		{
			name:          "wrong referer",
			refererHeader: "https://www.example.com.something.com:8080",
			allowed:       false,
		},
	}

	for _, testcase := range testcases {
		t.Run(testcase.name, func(t *testing.T) {
			userTokenClaims, _ := filter.iamClient.ValidateAndParseClaims("dummyToken")

			correctRequest := &restful.Request{
				Request: &http.Request{
					Header: map[string][]string{
						constant.Referer: {testcase.refererHeader},
					},
				},
			}

			actual := filter.validateRefererHeader(correctRequest, userTokenClaims, false)

			assert.Equal(t, testcase.allowed, actual)
		})
	}
}

// nolint:paralleltest
func TestValidateRefererHeaderWithoutSubdomain_RedirectURIIsIP(t *testing.T) {
	iamClient := &iam.MockClient{
		Healthy:     true,
		RedirectURI: "http://127.0.0.1",
	}
	filter := NewFilterWithOptions(iamClient, &FilterInitializationOptions{AllowSubdomainMatchRefererHeaderValidation: true})

	testcases := []struct {
		name          string
		refererHeader string
		allowed       bool
	}{
		{
			name:          "normal referer",
			refererHeader: "http://127.0.0.1",
			allowed:       true,
		},
		{
			name:          "normal referer",
			refererHeader: "http://127.0.0.1/admin/path",
			allowed:       true,
		},
		{
			name:          "normal referer",
			refererHeader: "http://subdomain.127.0.0.1",
			allowed:       true,
		},
		{
			name:          "normal referer",
			refererHeader: "http://subdomain.127.0.0.1/admin/path",
			allowed:       true,
		},
		{
			name:          "wrong referer",
			refererHeader: "https://example.net",
			allowed:       false,
		},
		{
			name:          "wrong referer",
			refererHeader: "http://127.0.0.2",
			allowed:       false,
		},
		{
			name:          "wrong referer",
			refererHeader: "http://127.0.0.1.2",
			allowed:       false,
		},
		{
			name:          "wrong referer",
			refererHeader: "http://subdomain.127.0.0.2",
			allowed:       false,
		},
	}

	for _, testcase := range testcases {
		t.Run(testcase.name, func(t *testing.T) {
			userTokenClaims, _ := filter.iamClient.ValidateAndParseClaims("dummyToken")

			correctRequest := &restful.Request{
				Request: &http.Request{
					Header: map[string][]string{
						constant.Referer: {testcase.refererHeader},
					},
				},
			}

			actual := filter.validateRefererHeader(correctRequest, userTokenClaims, false)

			assert.Equal(t, testcase.allowed, actual)
		})
	}
}

// nolint:paralleltest
func TestGetHost(t *testing.T) {
	testcases := []struct {
		name        string
		expected    string
		requestHost string
		URLHost     string
		scheme      string
	}{
		{
			name:        "not_absolute_url",
			requestHost: "host.example.com",
			URLHost:     "url.example.com",
			expected:    "host.example.com",
		},
		{
			name:        "not_absolute_url_with_port",
			requestHost: "host.example.com:80",
			URLHost:     "url.example.com",
			expected:    "host.example.com",
		},
		{
			name:        "absolute_url_without_port",
			requestHost: "host.example.com",
			URLHost:     "url.example.com",
			scheme:      "http",
			expected:    "url.example.com",
		},
	}

	for _, testcase := range testcases {
		t.Run(testcase.name, func(t *testing.T) {
			req := &http.Request{
				Host: testcase.requestHost,
				URL:  &url.URL{Host: testcase.URLHost, Scheme: testcase.scheme},
			}
			assert.Equal(t, testcase.expected, getHost(req))
		})
	}
}

func TestFilterInitializationOptionsFromEnv_SubdomainValidationEnabled(t *testing.T) {
	var options *FilterInitializationOptions

	os.Setenv("SUBDOMAIN_VALIDATION_ENABLED", "true")
	options = FilterInitializationOptionsFromEnv()
	assert.Equal(t, true, options.AllowSubdomainMatchRefererHeaderValidation)
	assert.Equal(t, true, options.SubdomainValidationEnabled)
	os.Unsetenv("SUBDOMAIN_VALIDATION_ENABLED")
}

func TestFilterInitializationOptionsFromEnv_SubdomainValidationDisabled(t *testing.T) {
	options := FilterInitializationOptionsFromEnv()
	assert.Equal(t, false, options.AllowSubdomainMatchRefererHeaderValidation)
	assert.Equal(t, false, options.SubdomainValidationEnabled)
}

func TestFilterInitializationOptionsFromEnv_SubdomainValidationExcludedNamespacesSet(t *testing.T) {
	var options *FilterInitializationOptions

	os.Setenv("SUBDOMAIN_VALIDATION_EXCLUDED_NAMESPACES", "foundations,foundations2,foundations3")
	options = FilterInitializationOptionsFromEnv()
	assert.Equal(t, []string{"foundations", "foundations2", "foundations3"}, options.SubdomainValidationExcludedNamespaces)

	os.Setenv("SUBDOMAIN_VALIDATION_EXCLUDED_NAMESPACES", "     foundations,foundations2,foundations3,,,    ")
	options = FilterInitializationOptionsFromEnv()
	assert.Equal(t, []string{"foundations", "foundations2", "foundations3"}, options.SubdomainValidationExcludedNamespaces)

	os.Unsetenv("SUBDOMAIN_VALIDATION_EXCLUDED_NAMESPACES")
}

func TestFilterInitializationOptionsFromEnv_SubdomainValidationExcludedNamespacesEmpty(t *testing.T) {
	var options *FilterInitializationOptions

	options = FilterInitializationOptionsFromEnv()
	assert.Empty(t, options.SubdomainValidationExcludedNamespaces)
}

func TestWithoutBannedTopics(t *testing.T) {
	timeNow := time.Now().UTC()
	futureBanTime := timeNow.Add(24 * time.Hour)
	pastBanTime := timeNow.Add(-24 * time.Hour)
	gameNamespace, publisherNamespace := "game", "publisher"

	testCases := []struct {
		name         string
		bannedTopics []string
		claims       *iam.JWTClaims
		wantErr      bool
		errMessage   restful.ServiceError
	}{
		{
			name:         "nil claims should pass",
			bannedTopics: []string{ChatBanTopic, MatchmakingBanTopic},
			claims:       nil,
			wantErr:      false,
		},
		{
			name:         "empty banned topics should pass",
			bannedTopics: []string{},
			claims: &iam.JWTClaims{
				Namespace:      gameNamespace,
				UnionNamespace: publisherNamespace,
				Bans: []iam.JWTBan{
					{Ban: ChatBanTopic, EndDate: futureBanTime, TargetedNamespace: gameNamespace},
				},
			},
			wantErr: false,
		},
		{
			name:         "non-matching ban topic should pass",
			bannedTopics: []string{MatchmakingBanTopic},
			claims: &iam.JWTClaims{
				Namespace:      gameNamespace,
				UnionNamespace: publisherNamespace,
				Bans: []iam.JWTBan{
					{Ban: ChatBanTopic, EndDate: futureBanTime, TargetedNamespace: gameNamespace},
				},
			},
			wantErr: false,
		},
		{
			name:         "expired ban should pass",
			bannedTopics: []string{ChatBanTopic},
			claims: &iam.JWTClaims{
				Namespace:      gameNamespace,
				UnionNamespace: publisherNamespace,
				Bans: []iam.JWTBan{
					{Ban: ChatBanTopic, EndDate: pastBanTime, TargetedNamespace: gameNamespace},
				},
			},
			wantErr: false,
		},
		{
			name:         "active matching ban should fail",
			bannedTopics: []string{MatchmakingBanTopic},
			claims: &iam.JWTClaims{
				Namespace:      gameNamespace,
				UnionNamespace: publisherNamespace,
				Bans: []iam.JWTBan{
					{Ban: MatchmakingBanTopic, EndDate: futureBanTime, TargetedNamespace: gameNamespace},
				},
			},
			wantErr: true,
			errMessage: respondError(http.StatusForbidden, ForbiddenAccess,
				fmt.Sprintf("access forbidden: user is banned due to %s ban until %s", MatchmakingBanTopic, futureBanTime.Format(time.RFC3339))),
		},
		{
			name:         "multiple bans with one active matching should fail",
			bannedTopics: []string{ChatBanTopic},
			claims: &iam.JWTClaims{
				Namespace:      gameNamespace,
				UnionNamespace: publisherNamespace,
				Bans: []iam.JWTBan{
					{Ban: "OTHER", EndDate: futureBanTime, TargetedNamespace: gameNamespace},
					{Ban: ChatBanTopic, EndDate: futureBanTime, TargetedNamespace: gameNamespace},
				},
			},
			wantErr: true,
			errMessage: respondError(http.StatusForbidden, ForbiddenAccess,
				fmt.Sprintf("access forbidden: user is banned due to %s ban until %s", ChatBanTopic, futureBanTime.Format(time.RFC3339))),
		},
		{
			name:         "active ban present, but bannedTopics does not match ban, should success",
			bannedTopics: []string{ChatBanTopic},
			claims: &iam.JWTClaims{
				Namespace:      gameNamespace,
				UnionNamespace: publisherNamespace,
				Bans: []iam.JWTBan{
					{Ban: MatchmakingBanTopic, EndDate: futureBanTime, TargetedNamespace: gameNamespace},
				},
			},
			wantErr: false,
		},
		{
			name:         "active ban present, but bannedTopics is empty, should succeed",
			bannedTopics: []string{""},
			claims: &iam.JWTClaims{
				Namespace:      gameNamespace,
				UnionNamespace: publisherNamespace,
				Bans: []iam.JWTBan{
					{Ban: MatchmakingBanTopic, EndDate: futureBanTime, TargetedNamespace: gameNamespace},
				},
			},
			wantErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			filterOpt := WithoutBannedTopics(tc.bannedTopics)
			err := filterOpt(&restful.Request{}, nil, tc.claims)

			if tc.wantErr {
				assert.Error(t, err)
				svcErr, ok := err.(restful.ServiceError)
				assert.True(t, ok)
				assert.Equal(t, http.StatusForbidden, svcErr.Code)
				assert.Equal(t, tc.errMessage.Message, svcErr.Message)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// nolint:paralleltest
func TestWithoutBannedTopics_TargetedNamespaceAndExpiry(t *testing.T) {
	now := time.Now().UTC()
	future := now.Add(24 * time.Hour)
	// EndDate equal to now (not in future)
	nowEnd := now

	testCases := []struct {
		name    string
		claims  *iam.JWTClaims
		banned  []string
		wantErr bool
	}{
		{
			name: "ban targets studio namespace -> allow chat on game namespace",
			claims: &iam.JWTClaims{
				Namespace:      "game",
				UnionNamespace: "publisher",
				Bans: []iam.JWTBan{
					{Ban: ChatBanTopic, TargetedNamespace: "publisher", EndDate: future},
				},
			},
			banned:  []string{ChatBanTopic},
			wantErr: false,
		},
		{
			name: "ban targets different namespace -> allow",
			claims: &iam.JWTClaims{
				Namespace:      "publisher",
				UnionNamespace: "publisher",
				Bans: []iam.JWTBan{
					{Ban: ChatBanTopic, TargetedNamespace: "game", EndDate: future},
				},
			},
			banned:  []string{ChatBanTopic},
			wantErr: false,
		},
		{
			name: "ban targets same namespace -> forbid",
			claims: &iam.JWTClaims{
				Namespace: "game",
				Bans: []iam.JWTBan{
					{Ban: ChatBanTopic, TargetedNamespace: "game", EndDate: future},
				},
			},
			banned:  []string{ChatBanTopic},
			wantErr: true,
		},
		{
			name: "targeted namespace case-insensitive match -> forbid",
			claims: &iam.JWTClaims{
				Namespace: "game",
				Bans: []iam.JWTBan{
					{Ban: MatchmakingBanTopic, TargetedNamespace: "GAME", EndDate: future},
				},
			},
			banned:  []string{MatchmakingBanTopic},
			wantErr: true,
		},
		{
			name: "ban end date equal to now -> allow (not before)",
			claims: &iam.JWTClaims{
				Namespace: "game",
				Bans: []iam.JWTBan{
					{Ban: ChatBanTopic, TargetedNamespace: "game", EndDate: nowEnd},
				},
			},
			banned:  []string{ChatBanTopic},
			wantErr: false,
		},
		{
			name: "ban end date equal to now -> allow (not before)",
			claims: &iam.JWTClaims{
				Namespace: "game",
				Bans: []iam.JWTBan{
					{Ban: ChatBanTopic, TargetedNamespace: "game", EndDate: nowEnd},
				},
			},
			banned:  []string{ChatBanTopic},
			wantErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			opt := WithoutBannedTopics(tc.banned)
			err := opt(&restful.Request{}, nil, tc.claims)
			if tc.wantErr {
				assert.Error(t, err)
				svcErr, ok := err.(restful.ServiceError)
				assert.True(t, ok)
				assert.Equal(t, http.StatusForbidden, svcErr.Code)
				// message should match respondError output for ForbiddenAccess
				expected := respondError(http.StatusForbidden, ForbiddenAccess,
					fmt.Sprintf("access forbidden: user is banned due to %s ban until %s", tc.claims.Bans[0].Ban, tc.claims.Bans[0].EndDate.Format(time.RFC3339)))
				assert.Equal(t, expected.Message, svcErr.Message)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// nolint:paralleltest
func TestWithoutBannedTopics_BannedTopicCaseSensitivity(t *testing.T) {
	now := time.Now().UTC().Add(24 * time.Hour)

	claims := &iam.JWTClaims{
		Namespace: "game",
		Bans: []iam.JWTBan{
			{Ban: ChatBanTopic, TargetedNamespace: "game", EndDate: now},
		},
	}

	t.Run("bannedTopics lower-case should match uppercase ban", func(t *testing.T) {
		opt := WithoutBannedTopics([]string{"chat"})
		err := opt(&restful.Request{}, nil, claims)
		assert.Error(t, err)
		svcErr, ok := err.(restful.ServiceError)
		assert.True(t, ok)
		assert.Equal(t, http.StatusForbidden, svcErr.Code)
	})

	t.Run("bannedTopics pascalCase should match uppercase ban", func(t *testing.T) {
		opt := WithoutBannedTopics([]string{"Chat"})
		err := opt(&restful.Request{}, nil, claims)
		assert.Error(t, err)
		svcErr, ok := err.(restful.ServiceError)
		assert.True(t, ok)
		assert.Equal(t, http.StatusForbidden, svcErr.Code)
	})
}

func TestWithValidSubscription_Success(t *testing.T) {
	testCases := []struct {
		name         string
		subscription string
		claims       *iam.JWTClaims
		wantErr      bool
		errMessage   restful.ServiceError
	}{
		{
			name:         "success - nil package on token claims - no subscription required",
			subscription: "",
			claims: &iam.JWTClaims{
				Subscriptions: nil,
			},
			wantErr: false,
		},
		{
			name:         "success - empty package on token claims - no subscription required",
			subscription: "",
			claims:       &iam.JWTClaims{},
			wantErr:      false,
		},
		{
			name:         "success - has subscription",
			subscription: MultiplayerPackage,
			claims: &iam.JWTClaims{
				Subscriptions: []string{FoundationsPackage, OnlinePackage, MultiplayerPackage, ExtendPackage},
			},
			wantErr: false,
		},
		{
			name:         "success - has subscription - upper case",
			subscription: strings.ToUpper(MultiplayerPackage),
			claims: &iam.JWTClaims{
				Subscriptions: []string{FoundationsPackage, OnlinePackage, MultiplayerPackage, ExtendPackage},
			},
			wantErr: false,
		},
		{
			name:         "success - has subscription - snake case",
			subscription: strings.ToTitle(MultiplayerPackage),
			claims: &iam.JWTClaims{
				Subscriptions: []string{FoundationsPackage, OnlinePackage, MultiplayerPackage, ExtendPackage},
			},
			wantErr: false,
		},
		{
			name:         "success - has subscription - lower case",
			subscription: strings.ToLower(MultiplayerPackage),
			claims: &iam.JWTClaims{
				Subscriptions: []string{FoundationsPackage, OnlinePackage, MultiplayerPackage, ExtendPackage},
			},
			wantErr: false,
		},
	}

	for idx, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			filterOpt := WithValidSubscription(tc.subscription)
			err := filterOpt(&restful.Request{}, iam.NewMockClient(), tc.claims)

			if tc.wantErr {
				assert.Error(t, err, fmt.Sprintf("%d - failed on test - %v", idx, tc.name))
				svcErr, ok := err.(restful.ServiceError)
				assert.True(t, ok)
				assert.Equal(t, http.StatusForbidden, svcErr.Code)
				assert.Contains(t, svcErr.Message, ErrorCodeMapping[InsufficientSubscription])
			} else {
				assert.NoError(t, err, fmt.Sprintf("%d - failed on test - %v", idx, tc.name))
			}
		})
	}
}

func TestWithValidSubscription_Failed(t *testing.T) {
	testCases := []struct {
		name         string
		subscription string
		claims       *iam.JWTClaims
		wantErr      bool
		errMessage   restful.ServiceError
	}{
		{
			name:         "failed - empty subscription check - claims has subscriptions",
			subscription: "",
			claims: &iam.JWTClaims{
				Subscriptions: []string{FoundationsPackage, OnlinePackage},
			},
			wantErr: true,
		},
		{
			name:         "failed - missing required package multiplayer",
			subscription: MultiplayerPackage,
			claims: &iam.JWTClaims{
				Subscriptions: []string{FoundationsPackage, OnlinePackage},
			},
			wantErr: true,
		},
		{
			name:         "failed - missing required package online",
			subscription: OnlinePackage,
			claims: &iam.JWTClaims{
				Subscriptions: []string{FoundationsPackage, MultiplayerPackage},
			},
			wantErr: true,
		},
		{
			name:         "failed - missing required package extend",
			subscription: ExtendPackage,
			claims: &iam.JWTClaims{
				Subscriptions: []string{FoundationsPackage, MultiplayerPackage},
			},
			wantErr: true,
		},
	}

	for idx, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			DevStackTraceable = true
			filterOpt := WithValidSubscription(tc.subscription)
			err := filterOpt(&restful.Request{}, iam.NewMockClient(), tc.claims)

			if tc.wantErr {
				assert.Error(t, err, fmt.Sprintf("%d - failed on test - %v", idx, tc.name))
				svcErr, ok := err.(restful.ServiceError)
				assert.True(t, ok)
				assert.Equal(t, http.StatusForbidden, svcErr.Code)
				assert.Contains(t, svcErr.Message, ErrorCodeMapping[InsufficientSubscription])
				if tc.subscription != "" {
					assert.Contains(t, svcErr.Message, tc.subscription)
				}
			} else {
				assert.NoError(t, err, fmt.Sprintf("%d - failed on test - %v", idx, tc.name))
			}
		})
	}
}
