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

import (
	"errors"
	"regexp"
	"strconv"
	"strings"

	"github.com/emicklei/go-restful/v3"
	"github.com/sirupsen/logrus"
)

// CrossOriginResourceSharing is used to create a Container Filter that implements CORS.
// Cross-origin resource sharing (CORS) is a mechanism that allows JavaScript on a web page
// to make XMLHttpRequests to another domain, not the domain the JavaScript originated from.
//
// http://en.wikipedia.org/wiki/Cross-origin_resource_sharing
// http://enable-cors.org/server.html
// https://web.dev/cross-origin-resource-sharing
type CrossOriginResourceSharing struct {
	ExposeHeaders  []string // list of exposed Headers
	AllowedHeaders []string // list of allowed Headers
	AllowedDomains []string // list of allowed values for Http Origin. An allowed value can be a regular expression to support subdomain matching. If empty all are allowed.
	AllowedMethods []string // list of allowed Methods
	MaxAge         int      // number of seconds that indicates how long the results of a preflight request can be cached.
	CookiesAllowed bool
	Container      *restful.Container
}

const (
	AllowedDomainsRegexPrefix = "re:"
)

// Filter is a filter function that implements the CORS flow
func (c CrossOriginResourceSharing) Filter(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
	origin := req.Request.Header.Get(restful.HEADER_Origin)
	if len(origin) == 0 {
		chain.ProcessFilter(req, resp)
		return
	}
	if !c.isOriginAllowed(origin) { // check whether this origin is allowed
		logrus.Debugf("HTTP Origin:%s is not part of %v", origin, c.AllowedDomains)
		chain.ProcessFilter(req, resp)
		return
	}

	if c.isPreflightRequest(req) {
		c.doPreflightRequest(req, resp)
		// return http 200 response, no body
		return
	}

	c.setOptionsHeaders(req, resp)
	chain.ProcessFilter(req, resp)
}

// isPreflightRequest will check if the request is a preflight request or not.
func (c *CrossOriginResourceSharing) isPreflightRequest(req *restful.Request) bool {
	if req.Request.Method == "OPTIONS" {
		if acrm := req.Request.Header.Get(restful.HEADER_AccessControlRequestMethod); acrm != "" {
			return true
		}
	}
	return false
}

// doPreflightRequest will set the necessary preflight headers into response header,
// e.g. Access-Control-Allow-Methods, Access-Control-Allow-Headers, Access-Control-Max-Age and all the default options headers (see setOptionsHeaders func)
func (c *CrossOriginResourceSharing) doPreflightRequest(req *restful.Request, resp *restful.Response) {
	acrm := req.Request.Header.Get(restful.HEADER_AccessControlRequestMethod)
	if !c.isValidAccessControlRequestMethod(acrm) {
		logrus.Debugf("Http header %s:%s is not in %v",
			restful.HEADER_AccessControlRequestMethod,
			acrm,
			c.AllowedMethods)
		return
	}
	acrhs := req.Request.Header.Get(restful.HEADER_AccessControlRequestHeaders)
	if len(acrhs) > 0 {
		for _, each := range strings.Split(acrhs, ",") {
			if !c.isValidAccessControlRequestHeader(strings.Trim(each, " ")) {
				logrus.Debugf("Http header %s:%s is not in %v",
					restful.HEADER_AccessControlRequestHeaders,
					acrhs,
					c.AllowedHeaders)
				return
			}
		}
	}
	resp.AddHeader(restful.HEADER_AccessControlAllowMethods, strings.Join(c.AllowedMethods, ","))
	resp.AddHeader(restful.HEADER_AccessControlAllowHeaders, acrhs)

	if c.MaxAge > 0 {
		resp.AddHeader(restful.HEADER_AccessControlMaxAge, strconv.Itoa(c.MaxAge))
	}

	c.setOptionsHeaders(req, resp)
}

// setOptionsHeaders will set option headers into response header,
// e.g. Access-Control-Allow-Origin, Access-Control-Expose-Headers, Access-Control-Allow-Credentials
func (c CrossOriginResourceSharing) setOptionsHeaders(req *restful.Request, resp *restful.Response) {
	origin := req.Request.Header.Get(restful.HEADER_Origin)
	resp.AddHeader(restful.HEADER_AccessControlAllowOrigin, origin)

	// some reference said that "Access-Control-Expose-Headers" should only be set for Actual request's response header (not Preflight request),
	// but we're keep it here to follow the current implementation from go-restful.
	if len(c.ExposeHeaders) > 0 {
		resp.AddHeader(restful.HEADER_AccessControlExposeHeaders, strings.Join(c.ExposeHeaders, ","))
	}

	if c.CookiesAllowed {
		resp.AddHeader(restful.HEADER_AccessControlAllowCredentials, "true")
	}
}

// isOriginAllowed will check if origin is allowed
func (c CrossOriginResourceSharing) isOriginAllowed(origin string) bool {
	if len(origin) == 0 {
		return false
	}
	if len(c.AllowedDomains) == 0 {
		return true
	}

	for _, domain := range c.AllowedDomains {
		if domain == origin || domain == "*" {
			return true
		}
		if strings.HasPrefix(domain, AllowedDomainsRegexPrefix) {
			pattern, err := getPattern(domain)
			if err != nil {
				return false
			}
			if pattern.MatchString(origin) {
				return true
			}
		}
	}

	return false
}

// isValidAccessControlRequestMethod will check if method is allowed
func (c CrossOriginResourceSharing) isValidAccessControlRequestMethod(method string) bool {
	for _, each := range c.AllowedMethods {
		if each == method {
			return true
		}
	}
	return false
}

// isValidAccessControlRequestHeader will check if header is allowed
func (c CrossOriginResourceSharing) isValidAccessControlRequestHeader(header string) bool {
	for _, each := range c.AllowedHeaders {
		if strings.ToLower(each) == strings.ToLower(header) {
			return true
		}
	}
	return false
}

func getPattern(str string) (*regexp.Regexp, error) {
	split := strings.Split(str, AllowedDomainsRegexPrefix)
	if len(split) < 2 {
		return nil, errors.New("pattern not found")
	}
	return regexp.Compile(split[1])
}
