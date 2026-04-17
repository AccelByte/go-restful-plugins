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
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	iam "github.com/AccelByte/iam-go-sdk/v2"
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

	// Dynamic CORS config support (optional - if ConfigServiceURL is set, dynamic config is enabled)
	ConfigServiceURL   string        // Base URL of justice-config-service (e.g. "http://justice-config-service/config"). If empty, static config is used.
	ConfigClient       ConfigClient  // Client for fetching namespace-scoped CORS config (set automatically from ConfigServiceURL on first request)
	ConfigFetchTimeout time.Duration // Per-request timeout for config service calls (default 200ms)
	IAMClient          iam.Client    // IAM client for obtaining bearer tokens to authenticate config service requests (optional)
	PublisherNamespace string        // Publisher namespace used to fetch subdomain extraction settings (CORS_SUBDOMAIN config key)

	// subdomain config is fetched lazily from the config service on first request and refreshed every subdomainConfigTTL
	subdomainMu       sync.Mutex
	subdomainConfig   *CORSSubdomainConfig
	subdomainLoaded   bool
	subdomainLoadedAt time.Time
}

// NewCrossOriginResourceSharing creates a new CORS filter with service-level default configuration.
// Parameters:
//   - configServiceURL: Base URL of justice-config-service (e.g. "http://justice-config-service/config").
//     Set to empty string to disable dynamic config and use static config only.
//   - iamClient: IAM client (iam.Client from iam-go-sdk/v2) for bearer-token authentication to config-service.
//     Required when configServiceURL is non-empty; pass nil only when configServiceURL is empty.
//   - publisherNamespace: Publisher namespace used to fetch subdomain extraction settings from the config service
//     (CORS_SUBDOMAIN config key). Subdomain extraction is disabled when empty or when configServiceURL is empty.
//   - allowedDomains: Service-level allowed origin domains (exact, wildcard, or re: regex patterns)
//   - allowedMethods: Service-level allowed HTTP methods
//   - allowedHeaders: Service-level allowed request headers
//   - exposeHeaders: Service-level exposed response headers
//   - cookiesAllowed: Whether credentials/cookies are allowed
//   - maxAge: Max age for preflight cache in seconds
func NewCrossOriginResourceSharing(
	configServiceURL string,
	iamClient iam.Client,
	publisherNamespace string,
	allowedDomains []string,
	allowedMethods []string,
	allowedHeaders []string,
	exposeHeaders []string,
	cookiesAllowed bool,
	maxAge int,
) (*CrossOriginResourceSharing, error) {
	if configServiceURL != "" && iamClient == nil {
		return nil, fmt.Errorf("cors: iamClient is required when configServiceURL is set")
	}
	return &CrossOriginResourceSharing{
		ConfigServiceURL:   configServiceURL,
		AllowedDomains:     allowedDomains,
		AllowedMethods:     allowedMethods,
		AllowedHeaders:     allowedHeaders,
		ExposeHeaders:      exposeHeaders,
		CookiesAllowed:     cookiesAllowed,
		MaxAge:             maxAge,
		IAMClient:          iamClient,
		PublisherNamespace: publisherNamespace,
	}, nil
}

// Filter is a filter function that implements the CORS flow
func (c *CrossOriginResourceSharing) Filter(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
	origin := req.Request.Header.Get(restful.HEADER_Origin)
	if len(origin) == 0 {
		chain.ProcessFilter(req, resp)
		return
	}

	// Initialize config client on first request if not already set
	if c.ConfigClient == nil {
		c.initConfigServiceClient()
	}

	// Try to fetch dynamic config if ConfigClient is available
	var config *MergedCORSConfig
	if c.ConfigClient != nil {
		config = c.getConfigWithDynamicResolution(req)
	}

	// Fall back to static config if dynamic resolution failed or is not available
	if config == nil {
		config = c.getStaticConfig()
	}

	if !c.isOriginAllowedWithConfig(config, origin) {
		logrus.Debugf("HTTP Origin:%s is not part of %v", origin, config.AllowedDomains)
		chain.ProcessFilter(req, resp)
		return
	}

	if c.isPreflightRequest(req) {
		c.doPreflightRequestWithConfig(req, resp, config)
		// return http 200 response, no body
		return
	}

	c.setOptionsHeadersWithConfig(req, resp, config)
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


// initConfigServiceClient initializes the config service client from ConfigServiceURL.
// If ConfigServiceURL is empty, dynamic config is disabled and static config is used.
// IAMClient is required when ConfigServiceURL is set; initialization is skipped with an error log if missing.
func (c *CrossOriginResourceSharing) initConfigServiceClient() {
	if c.ConfigServiceURL == "" {
		logrus.Debugf("ConfigServiceURL not set, CORS will use static configuration only")
		return
	}

	if c.IAMClient == nil {
		logrus.Errorf("cors: ConfigServiceURL is set but IAMClient is nil; " +
			"dynamic CORS config requires an IAM client — falling back to static config")
		return
	}

	c.ConfigClient = NewConfigClientWithIAM(c.ConfigServiceURL, 1*time.Minute, c.IAMClient, TransportConfig{})
	logrus.Infof("Initialized CORS config service client with URL: %s", c.ConfigServiceURL)
}

// getStaticConfig returns a MergedCORSConfig from the service-level static configuration.
func (c *CrossOriginResourceSharing) getStaticConfig() *MergedCORSConfig {
	return &MergedCORSConfig{
		AllowedDomains: c.AllowedDomains,
		AllowedHeaders: c.AllowedHeaders,
		AllowedMethods: c.AllowedMethods,
		ExposeHeaders:  c.ExposeHeaders,
		CookiesAllowed: c.CookiesAllowed,
		MaxAge:         c.MaxAge,
	}
}

// subdomainConfigTTL is the duration for which the fetched subdomain config is considered fresh.
const subdomainConfigTTL = time.Hour

// loadSubdomainConfig fetches the subdomain settings from the publisher namespace config.
// The result is cached for subdomainConfigTTL; on expiry the next request triggers a refresh.
// On failure the existing cached value is kept and subdomainLoadedAt is not updated, so the
// next request will retry immediately.
func (c *CrossOriginResourceSharing) loadSubdomainConfig() {
	if c.PublisherNamespace == "" {
		return
	}
	c.subdomainMu.Lock()
	defer c.subdomainMu.Unlock()
	if c.subdomainLoaded && time.Since(c.subdomainLoadedAt) < subdomainConfigTTL {
		return
	}
	cfg, err := c.ConfigClient.GetSubdomainConfig(c.PublisherNamespace)
	if err != nil {
		logrus.Errorf("cors: failed to fetch subdomain config for publisher namespace %q: %v", c.PublisherNamespace, err)
		return
	}
	c.subdomainConfig = cfg // nil is valid: means no config, subdomain extraction disabled
	c.subdomainLoaded = true
	c.subdomainLoadedAt = time.Now()
}

// getSubdomainSettings returns the subdomain extraction settings from the cached publisher namespace config.
func (c *CrossOriginResourceSharing) getSubdomainSettings() (enabled bool, baseDomain string) {
	c.subdomainMu.Lock()
	cfg := c.subdomainConfig
	c.subdomainMu.Unlock()
	if cfg == nil {
		return false, ""
	}
	return cfg.SubdomainEnabled, cfg.SubdomainBaseDomain
}

// getConfigWithDynamicResolution attempts to fetch and merge namespace-scoped config with static config.
// Returns nil if dynamic resolution fails (fallback to static config).
func (c *CrossOriginResourceSharing) getConfigWithDynamicResolution(req *restful.Request) *MergedCORSConfig {
	c.loadSubdomainConfig()
	subdomainEnabled, baseDomain := c.getSubdomainSettings()
	namespace := ExtractNamespace(req, subdomainEnabled, baseDomain)
	if namespace == "" {
		return nil
	}

	namespaceConfig, err := c.ConfigClient.GetCORSConfig(namespace)
	if err != nil {
		logrus.Errorf("Failed to fetch CORS config for namespace %s: %v", namespace, err)
		return nil
	}

	// Merge service and namespace configs
	serviceConfig := &CORSConfigValue{
		AllowedDomains: c.AllowedDomains,
		AllowedHeaders: c.AllowedHeaders,
		AllowedMethods: c.AllowedMethods,
		ExposeHeaders:  c.ExposeHeaders,
		CookiesAllowed: c.CookiesAllowed,
		MaxAge:         c.MaxAge,
	}

	return MergeConfigs(serviceConfig, namespaceConfig)
}

// isOriginAllowedWithConfig checks if origin is allowed according to the provided config.
func (c *CrossOriginResourceSharing) isOriginAllowedWithConfig(config *MergedCORSConfig, origin string) bool {
	if len(origin) == 0 {
		return false
	}
	if len(config.AllowedDomains) == 0 {
		return true
	}

	for _, domain := range config.AllowedDomains {
		pm, err := Compile(domain)
		if err != nil {
			logrus.Debugf("Invalid CORS domain pattern %q: %v", domain, err)
			continue
		}
		if pm.MatchOrigin(origin) {
			return true
		}
	}

	return false
}

// doPreflightRequestWithConfig handles preflight requests with the merged config.
func (c *CrossOriginResourceSharing) doPreflightRequestWithConfig(req *restful.Request, resp *restful.Response, config *MergedCORSConfig) {
	acrm := req.Request.Header.Get(restful.HEADER_AccessControlRequestMethod)
	if !c.isValidAccessControlRequestMethodWithConfig(config, acrm) {
		logrus.Debugf("Http header %s:%s is not in %v",
			restful.HEADER_AccessControlRequestMethod,
			acrm,
			config.AllowedMethods)
		return
	}
	acrhs := req.Request.Header.Get(restful.HEADER_AccessControlRequestHeaders)
	if len(acrhs) > 0 {
		for _, each := range strings.Split(acrhs, ",") {
			if !c.isValidAccessControlRequestHeaderWithConfig(config, strings.Trim(each, " ")) {
				logrus.Debugf("Http header %s:%s is not in %v",
					restful.HEADER_AccessControlRequestHeaders,
					acrhs,
					config.AllowedHeaders)
				return
			}
		}
	}
	resp.AddHeader(restful.HEADER_AccessControlAllowMethods, strings.Join(config.AllowedMethods, ","))
	resp.AddHeader(restful.HEADER_AccessControlAllowHeaders, acrhs)

	if config.MaxAge > 0 {
		resp.AddHeader(restful.HEADER_AccessControlMaxAge, strconv.Itoa(config.MaxAge))
	}

	c.setOptionsHeadersWithConfig(req, resp, config)
}

// setOptionsHeadersWithConfig sets CORS response headers using the merged config.
func (c *CrossOriginResourceSharing) setOptionsHeadersWithConfig(req *restful.Request, resp *restful.Response, config *MergedCORSConfig) {
	origin := req.Request.Header.Get(restful.HEADER_Origin)
	resp.AddHeader(restful.HEADER_AccessControlAllowOrigin, origin)

	if len(config.ExposeHeaders) > 0 {
		resp.AddHeader(restful.HEADER_AccessControlExposeHeaders, strings.Join(config.ExposeHeaders, ","))
	}

	if config.CookiesAllowed {
		resp.AddHeader(restful.HEADER_AccessControlAllowCredentials, "true")
	}
}

// isValidAccessControlRequestMethodWithConfig checks if method is allowed using the provided config.
func (c *CrossOriginResourceSharing) isValidAccessControlRequestMethodWithConfig(config *MergedCORSConfig, method string) bool {
	for _, each := range config.AllowedMethods {
		if each == method {
			return true
		}
	}
	return false
}

// isValidAccessControlRequestHeaderWithConfig checks if header is allowed using the provided config.
func (c *CrossOriginResourceSharing) isValidAccessControlRequestHeaderWithConfig(config *MergedCORSConfig, header string) bool {
	for _, each := range config.AllowedHeaders {
		if strings.ToLower(each) == strings.ToLower(header) {
			return true
		}
	}
	return false
}
