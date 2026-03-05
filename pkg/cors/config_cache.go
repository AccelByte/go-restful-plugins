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
	"time"

	"github.com/bluele/gcache"
)

const defaultCacheSize = 200

// ConfigCache is a loading cache for CORS configurations backed by gcache.
// When a namespace is not present, it automatically calls the configured loader function
// to fetch the config, then caches the result for the configured TTL duration.
type ConfigCache struct {
	gc gcache.Cache
}

// NewConfigCache creates a new LRU loading cache backed by gcache.
// The loader is called automatically on cache miss; its result is cached for the TTL duration.
// Use defaultCacheSize (200) entries maximum with LRU eviction policy.
func NewConfigCache(ttl time.Duration, loader func(string) (*CORSConfigValue, error)) *ConfigCache {
	gc := gcache.New(defaultCacheSize).
		LRU().
		Expiration(ttl).
		LoaderFunc(func(key interface{}) (interface{}, error) {
			return loader(key.(string))
		}).
		Build()
	return &ConfigCache{gc: gc}
}

// Get retrieves the CORS config for the given namespace.
// On a cache miss the loader is invoked automatically.
// Returns (nil, nil) when the namespace has no config (e.g. 404 from the config service).
func (cc *ConfigCache) Get(namespace string) (*CORSConfigValue, error) {
	v, err := cc.gc.Get(namespace)
	if err != nil {
		return nil, err
	}
	if v == nil {
		return nil, nil
	}
	return v.(*CORSConfigValue), nil
}

