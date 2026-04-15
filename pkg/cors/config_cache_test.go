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
	"sync"
	"testing"
	"time"
)

func TestNewConfigCache(t *testing.T) {
	loader := func(ns string) (*CORSConfigValue, error) {
		return &CORSConfigValue{AllowedDomains: []string{"https://example.com"}}, nil
	}
	cache := NewConfigCache(1*time.Minute, loader)
	if cache == nil {
		t.Fatal("NewConfigCache returned nil")
	}
}

func TestCacheLoaderCalledOnMiss(t *testing.T) {
	loadCount := 0
	config := &CORSConfigValue{AllowedDomains: []string{"https://example.com"}}
	loader := func(ns string) (*CORSConfigValue, error) {
		loadCount++
		return config, nil
	}
	cache := NewConfigCache(1*time.Minute, loader)

	result, err := cache.Get("test-ns")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("Expected config, got nil")
	}
	if loadCount != 1 {
		t.Errorf("Expected loader called once, got %d", loadCount)
	}
}

func TestCacheLoaderCalledOnceWithinTTL(t *testing.T) {
	loadCount := 0
	config := &CORSConfigValue{AllowedDomains: []string{"https://example.com"}}
	loader := func(ns string) (*CORSConfigValue, error) {
		loadCount++
		return config, nil
	}
	cache := NewConfigCache(1*time.Minute, loader)

	cache.Get("test-ns")
	cache.Get("test-ns")
	cache.Get("test-ns")

	if loadCount != 1 {
		t.Errorf("Expected loader called once (cached), got %d", loadCount)
	}
}

func TestCacheNilResult(t *testing.T) {
	// Loader returns nil (404 case — namespace has no config)
	loader := func(ns string) (*CORSConfigValue, error) {
		return nil, nil
	}
	cache := NewConfigCache(1*time.Minute, loader)

	result, err := cache.Get("nonexistent-ns")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result != nil {
		t.Error("Expected nil config for namespace with no config")
	}
}


func TestCacheConcurrentAccess(t *testing.T) {
	config := &CORSConfigValue{AllowedDomains: []string{"https://example.com"}}
	loader := func(ns string) (*CORSConfigValue, error) {
		return config, nil
	}
	cache := NewConfigCache(1*time.Minute, loader)

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				cache.Get("test-ns")
			}
		}()
	}
	wg.Wait()

	result, err := cache.Get("test-ns")
	if err != nil {
		t.Fatalf("Unexpected error after concurrent access: %v", err)
	}
	if result == nil {
		t.Error("Expected config after concurrent access")
	}
}

func TestCacheLoaderErrorNotCached(t *testing.T) {
	// gcache does NOT cache loader errors — each Get retries the loader
	loadCount := 0
	loader := func(ns string) (*CORSConfigValue, error) {
		loadCount++
		if loadCount < 3 {
			return nil, newConfigFetchError(ns, fmt.Errorf("temporary error"))
		}
		return &CORSConfigValue{AllowedDomains: []string{"https://example.com"}}, nil
	}
	cache := NewConfigCache(1*time.Minute, loader)

	_, err := cache.Get("test-ns")
	if err == nil {
		t.Error("Expected error on first call")
	}
	_, err = cache.Get("test-ns")
	if err == nil {
		t.Error("Expected error on second call")
	}
	result, err := cache.Get("test-ns")
	if err != nil {
		t.Fatalf("Expected success on third call, got error: %v", err)
	}
	if result == nil {
		t.Error("Expected config on third call")
	}
}
