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
	"net/url"
	"os"
	"testing"

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

			actual := filter.validateRefererHeader(correctRequest, userTokenClaims)
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

			actual := filter.validateRefererHeader(correctRequest, userTokenClaims)

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

			actual := filter.validateRefererHeader(correctRequest, userTokenClaims)
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

			actual := filter.validateRefererHeader(correctRequest, userTokenClaims)
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

			actual := filter.validateRefererHeader(correctRequest, userTokenClaims)
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

			actual := filter.validateRefererHeader(correctRequest, userTokenClaims)

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

			actual := filter.validateRefererHeader(correctRequest, userTokenClaims)

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

			actual := filter.validateRefererHeader(correctRequest, userTokenClaims)

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

			actual := filter.validateRefererHeader(correctRequest, userTokenClaims)

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
