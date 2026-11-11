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

// Package jaeger provides go-restful filters and helpers that create OpenTracing spans
// and report them with the Jaeger client.
//
// Deprecated: the Jaeger Go client and OpenTracing are no longer maintained upstream.
// Use OpenTelemetry tracing (go.opentelemetry.io/otel) instead. This package is kept
// for compatibility and gets no new features.
package jaeger
