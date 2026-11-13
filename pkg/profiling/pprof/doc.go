// Copyright 2026 AccelByte Inc
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

// Package pprof registers net/http/pprof handlers under a base path on http.DefaultServeMux.
//
// Deprecated: register pprof routes on the service's own go-restful container instead. This
// package registers them on the global http.DefaultServeMux, which panics when two services in
// one process use the same base path. It is kept for compatibility and gets no new features.
package pprof
