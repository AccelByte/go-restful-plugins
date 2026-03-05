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
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	iam "github.com/AccelByte/iam-go-sdk/v2"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// TransportConfig holds configurable HTTP transport settings for the config client.
// Zero values fall back to the defaults defined by defaultTransportConfig.
type TransportConfig struct {
	HTTPTimeout             time.Duration
	HTTPMaxIdleConns        int
	HTTPMaxIdleConnsPerHost int
	HTTPIdleConnTimeout     time.Duration
}

var defaultTransportConfig = TransportConfig{
	HTTPTimeout:             5 * time.Second,
	HTTPMaxIdleConns:        100,
	HTTPMaxIdleConnsPerHost: 10,
	HTTPIdleConnTimeout:     60 * time.Second,
}

func newHTTPClient(cfg TransportConfig) *http.Client {
	if cfg.HTTPTimeout == 0 {
		cfg.HTTPTimeout = defaultTransportConfig.HTTPTimeout
	}
	if cfg.HTTPMaxIdleConns == 0 {
		cfg.HTTPMaxIdleConns = defaultTransportConfig.HTTPMaxIdleConns
	}
	if cfg.HTTPMaxIdleConnsPerHost == 0 {
		cfg.HTTPMaxIdleConnsPerHost = defaultTransportConfig.HTTPMaxIdleConnsPerHost
	}
	if cfg.HTTPIdleConnTimeout == 0 {
		cfg.HTTPIdleConnTimeout = defaultTransportConfig.HTTPIdleConnTimeout
	}
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext(ctx, network, addr)
		},
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          cfg.HTTPMaxIdleConns,
		MaxIdleConnsPerHost:   cfg.HTTPMaxIdleConnsPerHost,
		IdleConnTimeout:       cfg.HTTPIdleConnTimeout,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	return &http.Client{
		Timeout:   cfg.HTTPTimeout,
		Transport: otelhttp.NewTransport(transport),
	}
}

// ConfigClient is the interface for fetching CORS configuration from a remote service.
type ConfigClient interface {
	GetCORSConfig(namespace string) (*CORSConfigValue, error)
	GetSubdomainConfig(publisherNamespace string) (*CORSSubdomainConfig, error)
}

// DefaultConfigClient is the HTTP transport implementation of ConfigClient.
// It uses a gcache loading cache: on a cache miss the cache automatically calls
// fetchFromService, so GetCORSConfig never needs to manage cache reads/writes manually.
type DefaultConfigClient struct {
	baseURL    string
	cache      *ConfigCache
	httpClient *http.Client
	iamClient  iam.Client // Optional IAM client for obtaining bearer tokens (can be nil)
}

// NewConfigClientWithIAM creates a config client with IAM bearer-token authentication.
// The iamClient is called before each config service request to obtain a fresh token.
// HTTP transport is configured via cfg; zero fields fall back to defaults.
// HTTP requests are traced via OpenTelemetry using otelhttp.
func NewConfigClientWithIAM(baseURL string, ttl time.Duration, iamClient iam.Client, cfg TransportConfig) *DefaultConfigClient {
	c := &DefaultConfigClient{
		baseURL:    baseURL,
		iamClient:  iamClient,
		httpClient: newHTTPClient(cfg),
	}
	c.cache = NewConfigCache(ttl, c.fetchFromService)
	return c
}

// GetCORSConfig returns the CORS configuration for the given namespace.
// The underlying gcache loading cache calls fetchFromService automatically on a miss.
// Returns (nil, nil) when the namespace has no config (graceful 404 fallback).
func (c *DefaultConfigClient) GetCORSConfig(namespace string) (*CORSConfigValue, error) {
	return c.cache.Get(namespace)
}

// GetSubdomainConfig fetches subdomain extraction settings for the publisher namespace
// from the CORS_SUBDOMAIN config key. Returns (nil, nil) when no config exists (404).
func (c *DefaultConfigClient) GetSubdomainConfig(publisherNamespace string) (*CORSSubdomainConfig, error) {
	url := fmt.Sprintf("%s/v1/admin/namespaces/%s/configs/CORS_SUBDOMAIN", c.baseURL, publisherNamespace)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, newConfigFetchError(publisherNamespace, err)
	}

	if c.iamClient != nil {
		token := c.iamClient.ClientToken()
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, newConfigFetchError(publisherNamespace, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, newConfigFetchError(publisherNamespace, fmt.Errorf("http status %d: %s", resp.StatusCode, string(body)))
	}

	var response ConfigServiceResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, newConfigFetchError(publisherNamespace, err)
	}

	if response.Value == "" {
		return nil, nil
	}

	var config CORSSubdomainConfig
	if err := json.Unmarshal([]byte(response.Value), &config); err != nil {
		return nil, newConfigFetchError(publisherNamespace, fmt.Errorf("failed to decode subdomain config value: %w", err))
	}

	return &config, nil
}

// fetchFromService fetches the CORS config from the justice-config-service.
// This function is passed as the loader to ConfigCache, so it is only called on cache misses.
// The API returns a nested JSON structure where the actual CORS config is in the "value" field as a JSON string.
// If an IAMClient is configured, a bearer token is included in the request.
func (c *DefaultConfigClient) fetchFromService(namespace string) (*CORSConfigValue, error) {
	url := fmt.Sprintf("%s/v1/admin/namespaces/%s/configs/CORS?includeParentConfig=studio,publisher", c.baseURL, namespace)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, newConfigFetchError(namespace, err)
	}

	if c.iamClient != nil {
		token := c.iamClient.ClientToken()
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, newConfigFetchError(namespace, err)
	}
	defer resp.Body.Close()

	// 404 means no config for this namespace — not an error, fallback to service defaults
	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	if resp.StatusCode >= http.StatusInternalServerError {
		body, _ := io.ReadAll(resp.Body)
		return nil, newConfigFetchError(namespace, fmt.Errorf("http status %d: %s", resp.StatusCode, string(body)))
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, newConfigFetchError(namespace, fmt.Errorf("http status %d: %s", resp.StatusCode, string(body)))
	}

	// Step 1: decode outer response
	var response ConfigServiceResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, newConfigFetchError(namespace, err)
	}

	// Step 2: unmarshal the "value" JSON string into CORSConfigValue
	if response.Value == "" {
		return nil, nil
	}

	var config CORSConfigValue
	if err := json.Unmarshal([]byte(response.Value), &config); err != nil {
		return nil, newConfigFetchError(namespace, fmt.Errorf("failed to decode CORS config value: %w", err))
	}

	return &config, nil
}
